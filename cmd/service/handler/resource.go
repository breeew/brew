package handler

import (
	"github.com/gin-gonic/gin"

	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/internal/response"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

type CreateResourceRequest struct {
	ID          string `json:"id" binding:"required"`
	Title       string `json:"title"`
	Cycle       *int   `json:"cycle"`
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
}

func (s *HttpSrv) CreateResource(c *gin.Context) {
	var (
		err error
		req CreateResourceRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	cycle := 0
	if req.Cycle != nil {
		cycle = *req.Cycle
	}

	spaceID, _ := v1.InjectSpaceID(c)
	err = v1.NewResourceLogic(c, s.Core).CreateResource(spaceID, req.ID, req.Title, req.Description, req.Prompt, cycle)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, nil)
}

type UpdateResourceRequest struct {
	ID          string `json:"id" binding:"required"`
	Title       string `json:"title"`
	Cycle       *int   `json:"cycle"`
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
}

func (s *HttpSrv) UpdateResource(c *gin.Context) {
	var (
		err error
		req UpdateResourceRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	cycle := 0
	if req.Cycle != nil {
		cycle = *req.Cycle
	}

	spaceID, _ := v1.InjectSpaceID(c)
	err = v1.NewResourceLogic(c, s.Core).Update(spaceID, req.ID, req.Title, req.Description, req.Prompt, cycle)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, nil)
}

func (s *HttpSrv) DeleteResource(c *gin.Context) {
	resourceID, _ := c.Params.Get("resourceid")

	spaceID, _ := v1.InjectSpaceID(c)
	err := v1.NewResourceLogic(c, s.Core).Delete(spaceID, resourceID)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, nil)
}

type ListResponse struct {
	List []types.Resource `json:"list"`
}

func (s *HttpSrv) ListResource(c *gin.Context) {
	spaceID, _ := v1.InjectSpaceID(c)
	list, err := v1.NewResourceLogic(c, s.Core).ListSpaceResources(spaceID)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, ListResponse{
		List: list,
	})
}

type GetResourceRequest struct {
	ID string `json:"id" binding:"required"`
}

func (s *HttpSrv) GetResource(c *gin.Context) {
	var (
		err error
		req GetResourceRequest
	)
	if err = utils.BindArgsWithGin(c, &req); err != nil {
		response.APIError(c, err)
		return
	}

	spaceID, _ := v1.InjectSpaceID(c)
	data, err := v1.NewResourceLogic(c, s.Core).GetResource(spaceID, req.ID)
	if err != nil {
		response.APIError(c, err)
		return
	}
	response.APISuccess(c, data)
}
