package srv

import (
	"net/http"

	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/mikespook/gorbac/v2"
)

const (
	// 定义角色ID
	RoleAdmin  = "role-admin"
	RoleEditor = "role-editor"
	RoleViewer = "role-viewer"

	// 定义权限ID
	PermissionAdmin = "admin"
	PermissionEdit  = "edit"
	PermissionView  = "view"

	BusinessUser = "bu"
	ClientUser   = "cu"
)

func SetupRBACSrv() *RBACSrv {
	// 创建一个新的RBAC实例
	rbac := gorbac.New()

	// 创建权限
	pAdmin := gorbac.NewStdPermission(PermissionAdmin)
	pEdit := gorbac.NewStdPermission(PermissionEdit)
	pView := gorbac.NewStdPermission(PermissionView)

	// 创建角色并分配权限
	roleAdmin := gorbac.NewStdRole(RoleAdmin)
	roleAdmin.Assign(pAdmin)

	roleEditor := gorbac.NewStdRole(RoleEditor)
	roleEditor.Assign(pEdit)

	roleViewer := gorbac.NewStdRole(RoleViewer)
	roleViewer.Assign(pView)

	// 将角色添加到RBAC实例
	rbac.Add(roleAdmin)
	rbac.Add(roleEditor)
	rbac.Add(roleViewer)

	// 设置角色继承关系
	rbac.SetParent(RoleEditor, RoleViewer) // 编辑者继承预览者的权限
	rbac.SetParent(RoleAdmin, RoleEditor)  // 管理者继承编辑者的权限
	return &RBACSrv{
		rbac: rbac,
	}
}

type RBACSrv struct {
	rbac *gorbac.RBAC
}

// checkPermission 检查角色是否有某权限
func (a *RBACSrv) CheckPermission(roleID, permissionID string) bool {
	return a.rbac.IsGranted(roleID, gorbac.NewStdPermission(permissionID), nil)
}

func (a *RBACSrv) CheckRoleAndPermission(roleID, role, permissionID string) bool {
	if roleID != role {
		return false
	}
	return a.CheckPermission(roleID, permissionID)
}

type RoleObject interface {
	GetUser() (string, error)
}

type fakeRoler struct {
	userID string
}

func (s *fakeRoler) GetUserId() string {
	return s.userID
}

func NewFakeRoler(userID string) *fakeRoler {
	return &fakeRoler{
		userID: userID,
	}
}

type LazyRoler struct {
	f      func() (string, error)
	userID string
}

func (s *LazyRoler) GetUser() (string, error) {
	if s.userID == "" {
		var err error
		if s.userID, err = s.f(); err != nil {
			return "", err
		}
	}
	return s.userID, nil
}

func NewRolerWithLazyload(f func() (string, error)) *LazyRoler {
	return &LazyRoler{
		f: f,
	}
}

type RoleUser interface {
	GetRole() string
	GetRoleType() string
	GetUser() string
}

// 如果是管理端用户，则只检测权限，如果是C端用户，则检测资源是否属于该用户
func (a *RBACSrv) Check(user RoleUser, obj RoleObject, permissionID string) *errors.CustomizedError {
	if user.GetRoleType() == BusinessUser {
		if !a.CheckPermission(user.GetRole(), permissionID) {
			return errors.New("RBACSrv.Check.BusinessUser", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
		}
	}
	resourceUser, err := obj.GetUser()
	if err != nil {
		return errors.Trace("RBACSrv.Check", err)
	}
	if user.GetUser() != resourceUser {
		return errors.New("RBACSrv.Check.ClientUser", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}
	return nil
}
