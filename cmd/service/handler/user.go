package handler

import (
	"github.com/gin-gonic/gin"

	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/internal/response"
)

type AccessLoginResponse struct {
	UserName string `json:"user_name"`
	UserID   string `json:"user_id"`
	Avatar   string `json:"avatar"`
}

func (s *HttpSrv) AccessLogin(c *gin.Context) {
	claims, _ := v1.InjectTokenClaim(c)

	user, err := v1.NewUserLogic(c, s.Core).GetUser(claims.Appid, claims.User)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, AccessLoginResponse{
		UserID:   user.ID,
		Avatar:   user.Avatar,
		UserName: user.Name,
	})
}
