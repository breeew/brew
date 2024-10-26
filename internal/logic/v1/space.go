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

type SpaceLogic struct {
	ctx  context.Context
	core *core.Core
	UserInfo
}

func NewSpaceLogic(ctx context.Context, core *core.Core) *SpaceLogic {
	l := &SpaceLogic{
		ctx:      ctx,
		core:     core,
		UserInfo: setupUserInfo(ctx, core),
	}

	return l
}

func (l *SpaceLogic) CreateUserSpace(title, desc string) (string, error) {
	user := l.GetUserInfo()
	spaceID := utils.GenRandomID()
	return spaceID, l.core.Store().Transaction(l.ctx, func(ctx context.Context) error {
		err := l.core.Store().SpaceStore().Create(ctx, types.Space{
			SpaceID:     spaceID,
			Title:       title,
			Description: desc,
			CreatedAt:   time.Now().Unix(),
		})
		if err != nil {
			return errors.New("SpaceLogic.CreateUserDefaultSpace.SpaceStore.Create", i18n.ERROR_INTERNAL, err)
		}

		err = l.core.Store().UserSpaceStore().Create(ctx, types.UserSpace{
			UserID:    user.User,
			SpaceID:   spaceID,
			Role:      srv.RoleAdmin,
			CreatedAt: time.Now().Unix(),
		})
		if err != nil {
			return errors.New("SpaceLogic.CreateUserDefaultSpace.UserSpaceStore.Create", i18n.ERROR_INTERNAL, err)
		}
		return nil
	})
}

func (l *SpaceLogic) SetUserSpaceRole(spaceID, userID, role string) error {
	switch role {
	case srv.RoleEditor:
	case srv.RoleViewer:
	default:
		return errors.New("SpaceLogic.SetUserSpaceRole.UnknownRole", i18n.ERROR_INVALIDARGUMENT, nil).Code(http.StatusBadRequest)
	}

	user := l.GetUserInfo()

	userSpace, err := l.core.Store().UserSpaceStore().GetUserSpaceRole(l.ctx, user.User, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("SpaceLogic.SetUserSpaceRole.UserSpaceStore.GetUserSpaceRole", i18n.ERROR_INTERNAL, err)
	}

	if userSpace == nil || !l.core.Srv().RBAC().CheckPermission(userSpace.Role, srv.PermissionAdmin) {
		return errors.New("SpaceLogic.SetUserSpaceRole.RBAC.CheckPermission", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}

	userExistRole, err := l.core.Store().UserSpaceStore().GetUserSpaceRole(l.ctx, userID, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("SpaceLogic.SetUserSpaceRole.UserSpaceStore.GetUserSpaceRole.settingUser", i18n.ERROR_INTERNAL, err)
	}

	if userExistRole == nil {
		err = l.core.Store().UserSpaceStore().Create(l.ctx, types.UserSpace{
			UserID:    userID,
			SpaceID:   spaceID,
			Role:      role,
			CreatedAt: time.Now().Unix(),
		})
		if err != nil {
			return errors.New("SpaceLogic.SetUserSpaceRole.UserSpaceStore.Create", i18n.ERROR_INTERNAL, err)
		}
	} else {
		if err = l.core.Store().UserSpaceStore().Update(l.ctx, userID, spaceID, role); err != nil {
			return errors.New("SpaceLogic.SetUserSpaceRole.UserSpaceStore.Update", i18n.ERROR_INTERNAL, err)
		}
	}

	return nil
}

func (l *SpaceLogic) ListSpaceUsers(spaceID string, page, pageSize uint64) ([]types.User, int64, error) {
	user := l.GetUserInfo()
	if !l.core.Srv().RBAC().CheckPermission(user.GetRole(), srv.PermissionAdmin) {
		return nil, 0, errors.New("SpaceLogic.ListSpaceUsers.CheckPermission", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}

	userIDs, err := l.core.Store().UserSpaceStore().ListSpaceUsers(l.ctx, spaceID)
	if err != nil {
		return nil, 0, errors.New("SpaceLogic.ListSpaceUsers.UserSpaceStore.ListSpaceUsers", i18n.ERROR_INTERNAL, err)
	}

	opts := types.ListUserOptions{
		IDs: userIDs,
	}
	list, err := l.core.Store().UserStore().ListUsers(l.ctx, opts, page, pageSize)
	if err != nil && err != sql.ErrNoRows {
		return nil, 0, errors.New("SpaceLogic.ListSpaceUsers.UserStore.ListUsers", i18n.ERROR_INTERNAL, err)
	}

	total, err := l.core.Store().UserStore().Total(l.ctx, opts)
	if err != nil {
		return nil, 0, errors.New("SpaceLogic.ListSpaceUsers.UserStore.Total", i18n.ERROR_INTERNAL, err)
	}

	return list, total, nil
}

func (l *SpaceLogic) UpdateSpace(spaceID, title, desc string) error {
	user := l.GetUserInfo()

	userSpace, err := l.core.Store().UserSpaceStore().GetUserSpaceRole(l.ctx, user.User, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("SpaceLogic.UpdateSpace.UserSpaceStore.GetUserSpaceRole", i18n.ERROR_INTERNAL, err)
	}

	if userSpace == nil || !l.core.Srv().RBAC().CheckPermission(userSpace.Role, srv.PermissionEdit) {
		return errors.New("SpaceLogic.UpdateSpace.RBAC.CheckPermission", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}

	if err = l.core.Store().SpaceStore().Update(l.ctx, spaceID, title, desc); err != nil {
		return errors.New("SpaceLogic.UpdateSpace.SpaceStore.Update", i18n.ERROR_INTERNAL, err)
	}

	return nil
}

func (l *SpaceLogic) LeaveSpace(spaceID string) error {
	user := l.GetUserInfo()

	userSpace, err := l.core.Store().UserSpaceStore().GetUserSpaceRole(l.ctx, user.User, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("SpaceLogic.LeaveSpace.UserSpaceStore.GetUserSpaceRole", i18n.ERROR_INTERNAL, err)
	}
	if userSpace == nil {
		return errors.New("SpaceLogic.LeaveSpace.userSpace.nil", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}

	//  delete all about space if this space only have one user
	list, err := l.core.Store().UserSpaceStore().ListSpaceUsers(l.ctx, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("SpaceLogic.LeaveSpace.UserSpaceStore.ListSpaceUsers", i18n.ERROR_INTERNAL, err)
	}

	if len(list) == 1 {
		if err = l.DeleteUserSpace(spaceID); err != nil {
			return errors.Trace("SpaceLogic.LeaveSpace", err)
		}
		return nil
	}

	if err = l.core.Store().UserSpaceStore().Delete(l.ctx, spaceID, user.User); err != nil {
		return errors.New("SpaceLogic.LeaveSpace.UserSpaceStore.Delete", i18n.ERROR_INTERNAL, err)
	}
	return nil
}

func (l *SpaceLogic) DeleteUserSpace(spaceID string) error {
	user := l.GetUserInfo()

	userSpace, err := l.core.Store().UserSpaceStore().GetUserSpaceRole(l.ctx, user.User, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("SpaceLogic.DeleteUserSpace.UserSpaceStore.GetUserSpaceRole", i18n.ERROR_INTERNAL, err)
	}

	if userSpace == nil || !l.core.Srv().RBAC().CheckPermission(userSpace.Role, srv.PermissionAdmin) {
		return errors.New("SpaceLogic.DeleteUserSpace.RBAC.CheckPermission", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}

	return l.core.Store().Transaction(l.ctx, func(ctx context.Context) error {
		if err := l.core.Store().UserSpaceStore().DeleteAll(ctx, spaceID); err != nil {
			return errors.New("SpaceLogic.DeleteUserSpace.UserSpaceStore.Delete", i18n.ERROR_INTERNAL, err)
		}

		if err := l.core.Store().SpaceStore().Delete(ctx, spaceID); err != nil {
			return errors.New("SpaceLogic.DeleteUserSpace.SpaceStore.Delete", i18n.ERROR_INTERNAL, err)
		}

		if err := l.core.Store().KnowledgeStore().DeleteAll(ctx, spaceID); err != nil {
			return errors.New("SpaceLogic.DeleteUserSpace.KnowledgeStore.DeleteAll", i18n.ERROR_INTERNAL, err)
		}

		if err := l.core.Store().VectorStore().DeleteAll(ctx, spaceID); err != nil {
			return errors.New("SpaceLogic.DeleteUserSpace.VectorStore.DeleteAll", i18n.ERROR_INTERNAL, err)
		}
		return nil
	})
}

func (l *SpaceLogic) ListUserSpace() ([]types.UserSpaceDetail, error) {
	user := l.GetUserInfo()
	list, err := l.core.Store().UserSpaceStore().List(l.ctx, types.ListUserSpaceOptions{
		UserID: user.User,
	}, 0, 0)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("SpaceLogic.ListUserSpace.UserSpaceStore.List", i18n.ERROR_INTERNAL, err)
	}

	var (
		spaceIDs     []string
		spaceRoleMap = make(map[string]string)
	)

	for _, v := range list {
		spaceIDs = append(spaceIDs, v.SpaceID)
		spaceRoleMap[v.SpaceID] = v.Role
	}

	spaceInfo, err := l.core.Store().SpaceStore().List(l.ctx, spaceIDs, 0, 0)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("SpaceLogic.ListUserSpace.SpaceStore.List", i18n.ERROR_INTERNAL, err)
	}

	var result []types.UserSpaceDetail
	for _, v := range spaceInfo {
		result = append(result, types.UserSpaceDetail{
			SpaceID:     v.SpaceID,
			UserID:      user.User,
			Title:       v.Title,
			Role:        spaceRoleMap[v.SpaceID],
			Description: v.Description,
			CreatedAt:   v.CreatedAt,
		})
	}

	return result, nil
}
