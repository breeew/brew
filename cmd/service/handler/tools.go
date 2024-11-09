package handler

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/internal/response"
	"github.com/starbx/brew-api/pkg/utils"
)

type ToolsReaderRequest struct {
	Endpoint string `json:"endpoint" form:"endpoint" binding:"required"`
}

func (s *HttpSrv) ToolsReader(c *gin.Context) {
	var (
		err error
		req ToolsReaderRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	res, err := v1.NewReaderLogic(c, s.Core).Reader(req.Endpoint)
	if err != nil {
		response.APIError(c, err)
		return
	}

	response.APISuccess(c, res)
}
