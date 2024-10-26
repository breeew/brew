package v1

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/starbx/brew-api/internal/core"
	"github.com/starbx/brew-api/internal/core/srv"
	"github.com/starbx/brew-api/pkg/errors"
	"github.com/starbx/brew-api/pkg/i18n"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

// logic for unlogin
type UserLogic struct {
	ctx  context.Context
	core *core.Core
}

func NewUserLogic(ctx context.Context, core *core.Core) *UserLogic {
	l := &UserLogic{
		ctx:  ctx,
		core: core,
	}

	return l
}

func (l *UserLogic) Register(appid, name, email, password string) (string, error) {
	salt := utils.RandomStr(10)
	userID := utils.GenRandomID()

	l.core.Store().Transaction(l.ctx, func(ctx context.Context) error {
		err := l.core.Store().UserStore().Create(ctx, types.User{
			ID:        userID,
			Appid:     appid,
			Name:      name,
			Email:     email,
			Avatar:    "",
			Salt:      salt,
			Source:    "platform",
			Password:  utils.GenUserPassword(salt, password),
			UpdatedAt: time.Now().Unix(),
			CreatedAt: time.Now().Unix(),
		})
		if err != nil {
			return errors.New("UserLogic.Register.UserStore.Create", i18n.ERROR_INTERNAL, err)
		}

		spaceID := utils.GenRandomID()
		err = l.core.Store().SpaceStore().Create(ctx, types.Space{
			SpaceID:     spaceID,
			Title:       "Main",
			Description: "default space",
			CreatedAt:   time.Now().Unix(),
		})
		if err != nil {
			return errors.New("UserLogic.Register.SpaceStore.Create", i18n.ERROR_INTERNAL, err)
		}

		err = l.core.Store().UserSpaceStore().Create(ctx, types.UserSpace{
			UserID:    userID,
			SpaceID:   spaceID,
			Role:      srv.RoleAdmin,
			CreatedAt: time.Now().Unix(),
		})
		if err != nil {
			return errors.New("UserLogic.Register.UserSpaceStore.Create", i18n.ERROR_INTERNAL, err)
		}
		return nil
	})

	return userID, nil
}

func (l *UserLogic) Login(appid, email, password string) (string, error) {
	user, err := l.core.Store().UserStore().GetByEmail(l.ctx, appid, email)
	if err != nil && err != sql.ErrNoRows {
		return "", errors.New("UserLogic.Login.UserStore.GetByEmail", i18n.ERROR_INTERNAL, err)
	}

	if user == nil || user.Password != utils.GenUserPassword(user.Salt, password) {
		return "", errors.New("UserLogic.Login.UserStore.GetByEmail", i18n.ERROR_INVALID_ACCOUNT, err).Code(http.StatusBadRequest)
	}

	accessToken := utils.MD5(user.ID + utils.GenRandomID())
	err = l.core.Store().AccessTokenStore().Create(l.ctx, types.AccessToken{
		UserID:    user.ID,
		Token:     accessToken,
		Version:   types.DEFAULT_ACCESS_TOKEN_VERSION,
		Info:      "login",
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		return "", errors.New("UserLogic.Login.AccessTokenStore.Login", i18n.ERROR_INTERNAL, err)
	}

	return accessToken, nil
}

func (l *UserLogic) GetUser(appid, id string) (*types.User, error) {
	user, err := l.core.Store().UserStore().GetUser(l.ctx, appid, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("AuthedUserLogin.GetUser.UserStore.GetUser", i18n.ERROR_INTERNAL, err)
	}

	if user == nil {
		return nil, errors.New("AuthedUserLogin.GetUser.UserStore.GetUser.nil", i18n.ERROR_INTERNAL, nil).Code(http.StatusNotFound)
	}

	return user, nil
}

type AuthedUserLogic struct {
	UserInfo
	ctx  context.Context
	core *core.Core
}

func NewAuthedUserLogic(ctx context.Context, core *core.Core) *AuthedUserLogic {
	l := &AuthedUserLogic{
		ctx:      ctx,
		core:     core,
		UserInfo: setupUserInfo(ctx, core),
	}

	return l
}
