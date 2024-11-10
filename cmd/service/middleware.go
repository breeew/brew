package service

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/breeew/brew-api/internal/core"
	v1 "github.com/breeew/brew-api/internal/logic/v1"
	"github.com/breeew/brew-api/internal/response"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
)

func I18n() gin.HandlerFunc {
	var allowList []string
	for k := range i18n.ALLOW_LANG {
		allowList = append(allowList, k)
	}
	l := i18n.NewLocalizer(allowList...)

	return response.ProvideResponseLocalizer(l)
}

const (
	ACCESS_TOKEN_HEADER_KEY = "X-Access-Token"
)

func AuthorizationFromQuery(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := c.Query("token")
		token, err := core.Store().AccessTokenStore().GetAccessToken(c, core.DefaultAppid(), tokenValue)
		if err != nil && err != sql.ErrNoRows {
			response.APIError(c, errors.New("checkAccessToken.AccessTokenStore.GetAccessToken", i18n.ERROR_INTERNAL, err))
			return
		}

		if token == nil || token.ExpiresAt < time.Now().Unix() {
			response.APIError(c, errors.New("checkAccessToken.token.check", i18n.ERROR_PERMISSION_DENIED, fmt.Errorf("nil token")).Code(http.StatusForbidden))
			return
		}

		claims, err := token.TokenClaims()
		if err != nil {
			response.APIError(c, errors.New("checkAccessToken.token.TokenClaims", i18n.ERROR_INVALID_TOKEN, err))
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

		if !matched {
			response.APIError(ctx, errors.New(tracePrefix, i18n.ERROR_PERMISSION_DENIED, err).Code(http.StatusForbidden))
			return
		}
	}
}

func checkAccessToken(ctx *gin.Context, core *core.Core) (bool, error) {
	tokenValue := ctx.GetHeader(ACCESS_TOKEN_HEADER_KEY)
	if tokenValue == "" {
		// try get
		// response.APIError(ctx, errors.New("middleware.AccessTokenVerify.GetHeader", i18n.ERROR_UNAUTHORIZED, nil))
		return false, errors.New("checkAccessToken.GetHeader.ACCESS_TOKEN_HEADER_KEY.nil", i18n.ERROR_UNAUTHORIZED, nil).Code(http.StatusUnauthorized)
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
