package v1

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/starbx/brew-api/internal/core"
	"github.com/starbx/brew-api/pkg/errors"
	"github.com/starbx/brew-api/pkg/i18n"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

type ResourceLogic struct {
	ctx  context.Context
	core *core.Core
	UserInfo
}

func NewResourceLogic(ctx context.Context, core *core.Core) *ResourceLogic {
	l := &ResourceLogic{
		ctx:      ctx,
		core:     core,
		UserInfo: setupUserInfo(ctx, core),
	}

	return l
}

func (l *ResourceLogic) CreateResource(spaceID, id, title, desc, prompt string, cycle int) error {
	if !utils.IsAlphabetic(id) {
		return errors.New("ResourceLogic.CreateResource.ID.IsAlphabetic", i18n.ERROR_INVALIDARGUMENT, fmt.Errorf("resource id is not alphabetic")).Code(http.StatusBadRequest)
	}
	if title == "" {
		title = id
	}

	if id == "knowledge" || title == "knowledge" {
		return errors.New("ResourceLogic.CreateResource.InvalidWord", i18n.ERROR_EXIST, nil).Code(http.StatusForbidden)
	}

	exist, err := l.core.Store().ResourceStore().GetResource(l.ctx, spaceID, id)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("ResourceLogic.CreateResource.ResourceStore.GetResource", i18n.ERROR_INTERNAL, err)
	}

	if exist != nil {
		return errors.New("ResourceLogic.CreateResource.exist", i18n.ERROR_EXIST, nil).Code(http.StatusBadRequest)
	}

	err = l.core.Store().ResourceStore().Create(l.ctx, types.Resource{
		ID:          id,
		UserID:      l.GetUserInfo().User,
		SpaceID:     spaceID,
		Title:       title,
		Description: desc,
		Prompt:      prompt,
		Cycle:       cycle,
		CreatedAt:   time.Now().Unix(),
	})
	if err != nil {
		return errors.New("ResourceLogic.CreateResource.ResourceStore.Create", i18n.ERROR_INTERNAL, err)
	}

	return nil
}

func (l *ResourceLogic) Delete(spaceID, id string) error {
	err := l.core.Store().ResourceStore().Delete(l.ctx, spaceID, id)
	if err != nil {
		return errors.New("ResourceLogic.Delete.ResourceStore.Delete", i18n.ERROR_INTERNAL, err)
	}
	return nil
}

func (l *ResourceLogic) Update(spaceID, id, title, desc, prompt string, cycle int) error {
	resources, err := l.core.Store().ResourceStore().ListResources(l.ctx, spaceID, types.NO_PAGING, types.NO_PAGING)
	if err != nil {
		return errors.New("ResourceLogic.Update.ResourceStore.ListResources", i18n.ERROR_INTERNAL, err)
	}

	for _, v := range resources {
		if v.ID != id && v.Title == title {
			return errors.New("ResourceLogic.Update.ResourceStore.ListResources", i18n.ERROR_TITLE_EXIST, nil).Code(http.StatusForbidden)
		}
	}

	err = l.core.Store().ResourceStore().Update(l.ctx, spaceID, id, title, desc, prompt, cycle)
	if err != nil {
		return errors.New("ResourceLogic.Update.ResourceStore.Update", i18n.ERROR_INTERNAL, err)
	}
	return nil
}

func (l *ResourceLogic) ListSpaceResources(spaceID string) ([]types.Resource, error) {
	list, err := l.core.Store().ResourceStore().ListResources(l.ctx, spaceID, 0, 0)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ResourceLogic.ListSpaceResources.ResourceStore.ListResources", i18n.ERROR_INTERNAL, err)
	}

	defaultKnowledgeResource := types.Resource{
		ID:      "knowledge",
		Title:   "knowledge",
		SpaceID: spaceID,
	}

	if len(list) == 0 {
		list = append(list, defaultKnowledgeResource)
	} else {
		list = append([]types.Resource{defaultKnowledgeResource}, list...)
	}

	return list, nil
}

func (l *ResourceLogic) GetResource(spaceID, id string) (*types.Resource, error) {
	data, err := l.core.Store().ResourceStore().GetResource(l.ctx, spaceID, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ResourceLogic.GetResource.ResourceStore.GetResource", i18n.ERROR_INTERNAL, err)
	}
	return data, nil
}
