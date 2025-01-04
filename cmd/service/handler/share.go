package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	v1 "github.com/breeew/brew-api/app/logic/v1"
	"github.com/breeew/brew-api/app/response"
	"github.com/breeew/brew-api/pkg/utils"
)

type CreateKnowledgeShareTokenRequest struct {
	EmbeddingURL string `json:"embedding_url" binding:"required"`
	KnowledgeID  string `json:"knowledge_id" binding:"required"`
}

type CreateKnowledgeShareTokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
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
	res, err := v1.NewManageShareLogic(c, s.Core).CreateKnowledgeShareToken(spaceID, req.KnowledgeID, req.EmbeddingURL)
	if err != nil {
		response.APIError(c, err)
		return
	}

	var shareURL string
	if s.Core.Cfg().Share.Domain != "" {
		shareURL = genKnowledgeShareURL(s.Core.Cfg().Share.Domain, res.Token)
	} else {
		shareURL = strings.ReplaceAll(req.EmbeddingURL, "{token}", res.Token)
	}

	response.APISuccess(c, CreateKnowledgeShareTokenResponse{
		Token: res.Token,
		URL:   shareURL,
	})
}

func genKnowledgeShareURL(domain, token string) string {
	return fmt.Sprintf("%s/s/k/%s", domain, token)
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
