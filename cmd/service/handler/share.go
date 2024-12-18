package handler

import (
	"github.com/gin-gonic/gin"

	v1 "github.com/breeew/brew-api/app/logic/v1"
	"github.com/breeew/brew-api/app/response"
	"github.com/breeew/brew-api/pkg/utils"
)

type CreateKnowledgeShareTokenRequest struct {
	KnowledgeID string `json:"knowledge_id" binding:"required"`
}

func (s *HttpSrv) CreateKnowledgeShareToken(c *gin.Context) {
	var (
		err error
		req CreateKnowledgeShareTokenRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	token, err := v1.NewManageShareLogic(c, s.Core).CreateKnowledgeShareToken(spaceID, req.KnowledgeID)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, token)
}

func (s *HttpSrv) GetKnowledgeByShareToken(c *gin.Context) {
	token, _ := c.Params.Get("token")

	res, err := v1.NewShareLogic(c, s.Core).GetKnowledgeByShareToken(token)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, res)
}
