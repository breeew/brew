package handler

import (
	v1 "github.com/breeew/brew-api/internal/logic/v1"
	"github.com/breeew/brew-api/internal/response"
	"github.com/breeew/brew-api/pkg/utils"
	"github.com/gin-gonic/gin"
)

type GenUploadKeyRequest struct {
	ObjectType string `json:"object_type" binding:"required"`
	Kind       string `json:"kind" binding:"required"`
	FileName   string `json:"file_name" binding:"required"`
}

// GenUploadKey
func (s *HttpSrv) GenUploadKey(c *gin.Context) {

	var (
		err error
		req GenUploadKeyRequest
	)

	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	logic := v1.NewUploadLogic(c, s.Core)
	result, err := logic.GenClientUploadKey(req.ObjectType, req.Kind, req.FileName)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, result)
}
