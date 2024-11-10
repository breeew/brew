package v1

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/breeew/brew-api/internal/core"
	"github.com/breeew/brew-api/internal/core/srv"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/breeew/brew-api/pkg/security"
)

type _userInfo struct {
	ctx     context.Context
	core    *core.Core
	u       *security.TokenClaims
	checker func(roler srv.RoleObject, permit string) bool
}

func (u *_userInfo) GetUserInfo() security.TokenClaims {
	return *u.u
}

func (u *_userInfo) Identification(roler srv.RoleObject, permission string) error {
	if err := u.core.Srv().RBAC().Check(u.GetUserInfo(), roler, permission); err != nil {
		return err
	}
	return nil
}

// 通过eventid获取该event对应的用户id

func (u *_userInfo) lazyRolerFromKnowledgeID(spaceID, id string) *srv.LazyRoler {
	return srv.NewRolerWithLazyload(func() (string, error) {
		e, err := u.core.Store().KnowledgeStore().GetKnowledge(u.ctx, spaceID, id)
		if err != nil && err != sql.ErrNoRows {
			slog.Error("Failed to get userID by event", slog.String("error", errors.New("lazyRoler", "error.internal", err).Error()))
			return "", errors.New("_userInfo.RolerWithLazyload", i18n.ERROR_INTERNAL, err)
		}
		return e.UserID, nil
	})
}

func setupUserInfo(ctx context.Context, core *core.Core) UserInfo {
	userInfo, ok := InjectTokenClaim(ctx)
	if !ok {
		slog.Error("Not found user in context", slog.String("component", "logic.v1.setupUserInfo"))
		userInfo = security.TokenClaims{}
	}
	return &_userInfo{
		u:    &userInfo,
		core: core,
	}
}

type UserInfo interface {
	GetUserInfo() security.TokenClaims
	Identification(roler srv.RoleObject, permission string) error
	lazyRolerFromKnowledgeID(spaceID, id string) *srv.LazyRoler
}
