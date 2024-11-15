package handler

import (
	"net/http"

	"github.com/breeew/brew-api/internal/response"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/gin-gonic/gin"
)

// UploadFileHandler
func (s *HttpSrv) UploadFileHandler(c *gin.Context) {
	file, err := c.FormFile("file") // "file" 是表单中字段的名称
	if err != nil {
		response.APIError(c, errors.New("API.UploadFileHandler.FromFile", i18n.ERROR_INVALIDARGUMENT, err).Code(http.StatusBadRequest))
		return
	}

	file = file

	// TODO: uploadLogic
}
