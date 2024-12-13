package handler

import (
	"github.com/gin-gonic/gin"

	v1 "github.com/breeew/brew-api/app/logic/v1"
	"github.com/breeew/brew-api/app/response"
	"github.com/breeew/brew-api/pkg/utils"
)

type AccessLoginResponse struct {
	UserName    string `json:"user_name"`
	UserID      string `json:"user_id"`
	Avatar      string `json:"avatar"`
	Email       string `json:"email"`
	ServiceMode string `json:"service_mode"`
}

func (s *HttpSrv) AccessLogin(c *gin.Context) {
	claims, _ := v1.InjectTokenClaim(c)

	user, err := v1.NewUserLogic(c, s.Core).GetUser(claims.Appid, claims.User)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, AccessLoginResponse{
		UserID:      user.ID,
		Avatar:      user.Avatar,
		UserName:    user.Name,
		Email:       user.Email,
		ServiceMode: s.Core.Plugins.Name(),
	})
}

type UpdateUserProfileRequest struct {
	UserName string `json:"user_name" form:"user_name" binding:"required,max=32"`
	Email    string `json:"email" form:"email" binding:"required,email"`
}

func (s *HttpSrv) UpdateUserProfile(c *gin.Context) {
	var (
		err error
		req UpdateUserProfileRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	err = v1.NewAuthedUserLogic(c, s.Core).UpdateUserProfile(req.UserName, req.Email)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, nil)
}
