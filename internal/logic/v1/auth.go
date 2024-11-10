package v1

import (
	"context"
	"database/sql"

	"github.com/breeew/brew-api/internal/core"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/breeew/brew-api/pkg/types"
	"github.com/breeew/brew-api/pkg/utils"
)

type AuthLogic struct {
	ctx  context.Context
	core *core.Core
}

// 用户登录后可以创建多个 spaceid, 用户申请的token可以设置访问范围？
// 这个token能访问全部spaceid, 或这个token只能访问某些spaceid？
// 只从ToC的角度来看，token默认就可以访问他所代表用户的全部spaceid
func NewAuthLogic(ctx context.Context, core *core.Core) *AuthLogic {
	l := &AuthLogic{
		ctx:  ctx,
		core: core,
	}

	return l
}

func (l *AuthLogic) GetAccessTokenDetail(appid, token string) (*types.AccessToken, error) {
	data, err := l.core.Store().AccessTokenStore().GetAccessToken(l.ctx, appid, token)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("AuthLogic.GetAccessTokenDetail.AccessTokenStore.GetAccessToken", i18n.ERROR_INTERNAL, err)
	}

	return data, nil
}

func (l *AuthLogic) GenAccessToken(appid, desc, userID string, expiresAt int64) (string, error) {
	tokenStore := l.core.Store().AccessTokenStore()
REGEN:
	accessToken := utils.RandomStr(100)
	exist, err := tokenStore.GetAccessToken(l.ctx, appid, accessToken)
	if err != nil && err != sql.ErrNoRows {
		return "", errors.New("AuthLogic.GenNewAccessToken.GetAccessToken", i18n.ERROR_INTERNAL, err)
	}

	if exist != nil {
		// TODO: limit
		goto REGEN
	}

	err = tokenStore.Create(l.ctx, types.AccessToken{
		Appid:     appid,
		UserID:    userID,
		Version:   types.DEFAULT_ACCESS_TOKEN_VERSION,
		Token:     accessToken,
		ExpiresAt: expiresAt,
		Info:      desc,
	})

	if err != nil {
		return "", errors.New("AuthLogic.GenNewAccessToken.Create", i18n.ERROR_INTERNAL, err)
	}

	return accessToken, nil
}
