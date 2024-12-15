package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/breeew/brew-api/app/core"
	v1 "github.com/breeew/brew-api/app/logic/v1"
	"github.com/breeew/brew-api/app/response"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/breeew/brew-api/pkg/security"
	"github.com/breeew/brew-api/pkg/types"
	"github.com/breeew/brew-api/pkg/utils"
)

func I18n() gin.HandlerFunc {
	var allowList []string
	for k := range i18n.ALLOW_LANG {
		allowList = append(allowList, k)
	}
	l := i18n.NewLocalizer(allowList...)

	return response.ProvideResponseLocalizer(l)
}

// AcceptLanguage 目前服务端支持 en: English, zh-CN: 简体中文
func AcceptLanguage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		lang := ctx.Request.Header.Get("Accept-Language")
		if lang == "" {
			ctx.Set(v1.LANGUAGE_KEY, types.LANGUAGE_EN_KEY)
			return
		}

		res := utils.ParseAcceptLanguage(lang)
		if len(res) == 0 {
			ctx.Set(v1.LANGUAGE_KEY, types.LANGUAGE_EN_KEY)
			return
		}

		ctx.Set(v1.LANGUAGE_KEY, lo.If[string](strings.Contains(res[0].Tag, "zh"), types.LANGUAGE_CN_KEY).Else(types.LANGUAGE_EN_KEY))
	}
}

const (
	ACCESS_TOKEN_HEADER_KEY = "X-Access-Token"
	AUTH_TOKEN_HEADER_KEY   = "X-Authorization"
)

func AuthorizationFromQuery(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := c.Query("token")
		tokenType := c.Query("token-type")
		if tokenType == "atuhorization" {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			tokenMetaStr, err := core.Plugins.Cache().Get(ctx, fmt.Sprintf("user:token:%s", utils.MD5(tokenValue)))
			if err != nil {
				response.APIError(c, errors.New("AuthorizationFromQuery.GetFromCache", i18n.ERROR_INTERNAL, err))
				return
			}

			var tokenMeta types.UserTokenMeta
			if err := json.Unmarshal([]byte(tokenMetaStr), &tokenMeta); err != nil {
				response.APIError(c, errors.New("AuthorizationFromQuery.GetFromCache.json.Unmarshal", i18n.ERROR_INTERNAL, err))
				return
			}

			c.Set(v1.TOKEN_CONTEXT_KEY, security.NewTokenClaims(tokenMeta.Appid, "brew", tokenMeta.UserID, "", tokenMeta.ExpireAt))
			return
		}

		token, err := core.Store().AccessTokenStore().GetAccessToken(c, core.DefaultAppid(), tokenValue)
		if err != nil && err != sql.ErrNoRows {
			response.APIError(c, errors.New("AuthorizationFromQuery.AccessTokenStore.GetAccessToken", i18n.ERROR_INTERNAL, err))
			return
		}

		if token == nil || token.ExpiresAt < time.Now().Unix() {
			response.APIError(c, errors.New("AuthorizationFromQuery.token.check", i18n.ERROR_PERMISSION_DENIED, fmt.Errorf("nil token")).Code(http.StatusForbidden))
			return
		}

		claims, err := token.TokenClaims()
		if err != nil {
			response.APIError(c, errors.New("AuthorizationFromQuery.token.TokenClaims", i18n.ERROR_INVALID_TOKEN, err))
			return
		}

		c.Set(v1.TOKEN_CONTEXT_KEY, *claims)
	}
}

func Authorization(core *core.Core) gin.HandlerFunc {
	tracePrefix := "middleware.TryGetAccessToken"
	return func(ctx *gin.Context) {
		matched, err := checkAccessToken(ctx, core)
		if err != nil {
			response.APIError(ctx, errors.Trace(tracePrefix, err))
			return
		}

		if matched {
			return
		}

		if matched, err = checkAuthToken(ctx, core); err != nil || !matched {
			response.APIError(ctx, errors.New(tracePrefix, i18n.ERROR_PERMISSION_DENIED, err).Code(http.StatusForbidden))
		}
	}
}

func checkAccessToken(ctx *gin.Context, core *core.Core) (bool, error) {
	tokenValue := ctx.GetHeader(ACCESS_TOKEN_HEADER_KEY)
	if tokenValue == "" {
		// try get
		// errors.New("checkAccessToken.GetHeader.ACCESS_TOKEN_HEADER_KEY.nil", i18n.ERROR_UNAUTHORIZED, nil).Code(http.StatusUnauthorized)
		return false, nil
	}

	appid := core.DefaultAppid()

	token, err := core.Store().AccessTokenStore().GetAccessToken(ctx, appid, tokenValue)
	if err != nil && err != sql.ErrNoRows {
		return false, errors.New("checkAccessToken.AccessTokenStore.GetAccessToken", i18n.ERROR_INTERNAL, err)
	}

	if token == nil || token.ExpiresAt < time.Now().Unix() {
		return false, errors.New("checkAccessToken.token.check", i18n.ERROR_PERMISSION_DENIED, fmt.Errorf("nil token")).Code(http.StatusForbidden)
	}

	claims, err := token.TokenClaims()
	if err != nil {
		return false, errors.New("checkAccessToken.token.TokenClaims", i18n.ERROR_INVALID_TOKEN, err)
	}

	ctx.Set(v1.TOKEN_CONTEXT_KEY, *claims)
	return true, nil
}

func checkAuthToken(c *gin.Context, core *core.Core) (bool, error) {
	tokenValue := c.GetHeader(AUTH_TOKEN_HEADER_KEY)
	if tokenValue == "" {
		// try get
		// errors.New("checkAuthToken.GetHeader.AUTH_TOKEN_HEADER_KEY.nil", i18n.ERROR_UNAUTHORIZED, nil).Code(http.StatusUnauthorized)
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	tokenMetaStr, err := core.Plugins.Cache().Get(ctx, fmt.Sprintf("user:token:%s", utils.MD5(tokenValue)))
	if err != nil {
		return false, errors.New("AuthorizationFromQuery.GetFromCache", i18n.ERROR_INTERNAL, err)
	}

	var tokenMeta types.UserTokenMeta
	if err := json.Unmarshal([]byte(tokenMetaStr), &tokenMeta); err != nil {
		return false, errors.New("AuthorizationFromQuery.GetFromCache.json.Unmarshal", i18n.ERROR_INTERNAL, err)
	}

	c.Set(v1.TOKEN_CONTEXT_KEY, security.NewTokenClaims(tokenMeta.Appid, "brew", tokenMeta.UserID, "", tokenMeta.ExpireAt))
	return true, nil
}

func VerifySpaceIDPermission(core *core.Core, permission string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		spaceID, _ := ctx.Params.Get("spaceid")

		claims, _ := v1.InjectTokenClaim(ctx)

		result, err := core.Store().UserSpaceStore().GetUserSpaceRole(ctx, claims.User, spaceID)
		if err != nil && err != sql.ErrNoRows {
			response.APIError(ctx, errors.New("middleware.VerifySpaceIDPermission.UserSpaceStore.GetUserSpaceRole", i18n.ERROR_INTERNAL, err))
			return
		}

		if result == nil {
			response.APIError(ctx, errors.New("middleware.VerifySpaceIDPermission.UserSpaceStore.GetUserSpaceRole.nil", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden))
			return
		}

		claims.Fields["role"] = result.Role

		if !core.Srv().RBAC().CheckPermission(result.Role, permission) {
			response.APIError(ctx, errors.New("middleware.VerifySpaceIDPermission.CheckPermission", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden))
			return
		}

		ctx.Set(v1.SPACEID_CONTEXT_KEY, spaceID)
	}
}

func Cors(c *gin.Context) {
	method := c.Request.Method
	origin := c.Request.Header.Get("Origin")
	if origin != "" {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-Access-Token")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	if method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
	}
	c.Next()
}

func UseLimit(core *core.Core, operation string, genKeyFunc func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !core.UseLimiter(genKeyFunc(c), operation, 4).Allow() {
			response.APIError(c, errors.New("middleware.limiter", i18n.ERROR_TOO_MANY_REQUESTS, nil).Code(http.StatusTooManyRequests))
		}
	}
}
