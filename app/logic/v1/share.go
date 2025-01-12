package v1

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/breeew/brew-api/app/core"
	"github.com/breeew/brew-api/app/core/srv"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/breeew/brew-api/pkg/types"
	"github.com/breeew/brew-api/pkg/utils"
)

type ManageShareLogic struct {
	ctx  context.Context
	core *core.Core
	UserInfo
}

func NewManageShareLogic(ctx context.Context, core *core.Core) *ManageShareLogic {
	l := &ManageShareLogic{
		ctx:      ctx,
		core:     core,
		UserInfo: SetupUserInfo(ctx, core),
	}

	return l
}

type CreateKnowledgeShareTokenResult struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func (l *ManageShareLogic) CreateKnowledgeShareToken(spaceID, knowledgeID, embeddingURL string) (CreateKnowledgeShareTokenResult, error) {
	res := CreateKnowledgeShareTokenResult{}

	userSpaceRole, err := l.core.Store().UserSpaceStore().GetUserSpaceRole(l.ctx, l.GetUserInfo().User, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return res, errors.New("ManageShareLogic.CreateKnowledgeShareToken.UserSpaceStore.GetUserSpaceRole", i18n.ERROR_INTERNAL, err)
	}

	if userSpaceRole == nil || userSpaceRole.Role != srv.RoleAdmin {
		return res, errors.New("ManageShareLogic.CreateKnowledgeShareToken.Role.Check", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}

	link, err := l.core.Store().ShareTokenStore().Get(l.ctx, types.SHARE_TYPE_KNOWLEDGE, spaceID, knowledgeID)
	if err != nil && err != sql.ErrNoRows {
		return res, errors.New("ManageShareLogic.CreateKnowledgeShareToken.ShareTokenStore.Get", i18n.ERROR_INTERNAL, err)
	}

	if link != nil {
		if link.ExpireAt < time.Now().AddDate(0, 0, -1).Unix() {
			// update link expire time
			if err = l.core.Store().ShareTokenStore().UpdateExpireTime(l.ctx, link.ID, time.Now().AddDate(0, 0, 7).Unix()); err != nil {
				slog.Error("Failed to update share link expire time", slog.String("error", err.Error()), slog.String("space_id", spaceID),
					slog.String("knowledge_id", knowledgeID))
			}
		}

		res.Token = link.Token
		res.URL = link.EmbeddingURL
		return res, nil
	}

	res.Token = utils.MD5(fmt.Sprintf("%s_%s_%d", spaceID, knowledgeID, utils.GenUniqID()))
	res.URL = strings.ReplaceAll(embeddingURL, "{token}", res.Token)

	err = l.core.Store().ShareTokenStore().Create(l.ctx, &types.ShareToken{
		Appid:        l.GetUserInfo().Appid,
		Type:         types.SHARE_TYPE_KNOWLEDGE,
		SpaceID:      spaceID,
		ObjectID:     knowledgeID,
		Token:        res.Token,
		ShareUserID:  l.GetUserInfo().User,
		EmbeddingURL: res.URL,
		ExpireAt:     time.Now().AddDate(0, 0, 7).Unix(),
		CreatedAt:    time.Now().Unix(),
	})
	if err != nil {
		return res, errors.New("ManageShareLogic.CreateKnowledgeShareToken.ShareTokenStore.Create", i18n.ERROR_INTERNAL, err)
	}

	return res, nil
}

type CreateSessionShareTokenResult struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func (l *ManageShareLogic) CreateSessionShareToken(spaceID, sessionID, embeddingURL string) (CreateSessionShareTokenResult, error) {
	res := CreateSessionShareTokenResult{}

	userSpaceRole, err := l.core.Store().UserSpaceStore().GetUserSpaceRole(l.ctx, l.GetUserInfo().User, spaceID)
	if err != nil && err != sql.ErrNoRows {
		return res, errors.New("ManageShareLogic.CreateSessionShareToken.UserSpaceStore.GetUserSpaceRole", i18n.ERROR_INTERNAL, err)
	}

	if userSpaceRole == nil || userSpaceRole.Role != srv.RoleAdmin {
		return res, errors.New("ManageShareLogic.CreateSessionShareToken.Role.Check", i18n.ERROR_PERMISSION_DENIED, nil).Code(http.StatusForbidden)
	}

	link, err := l.core.Store().ShareTokenStore().Get(l.ctx, types.SHARE_TYPE_SESSION, spaceID, sessionID)
	if err != nil && err != sql.ErrNoRows {
		return res, errors.New("ManageShareLogic.CreateSessionShareToken.ShareTokenStore.Get", i18n.ERROR_INTERNAL, err)
	}

	if link != nil {
		if link.ExpireAt < time.Now().AddDate(0, 0, -1).Unix() {
			// update link expire time
			if err = l.core.Store().ShareTokenStore().UpdateExpireTime(l.ctx, link.ID, time.Now().AddDate(0, 0, 7).Unix()); err != nil {
				slog.Error("Failed to update share link expire time", slog.String("error", err.Error()), slog.String("space_id", spaceID),
					slog.String("knowledge_id", sessionID))
			}
		}

		res.Token = link.Token
		res.URL = link.EmbeddingURL
		return res, nil
	}

	res.Token = utils.MD5(fmt.Sprintf("%s_%s_%d", spaceID, sessionID, utils.GenUniqID()))
	res.URL = strings.ReplaceAll(embeddingURL, "{token}", res.Token)

	err = l.core.Store().ShareTokenStore().Create(l.ctx, &types.ShareToken{
		Appid:        l.GetUserInfo().Appid,
		Type:         types.SHARE_TYPE_SESSION,
		SpaceID:      spaceID,
		ObjectID:     sessionID,
		Token:        res.Token,
		ShareUserID:  l.GetUserInfo().User,
		EmbeddingURL: res.URL,
		ExpireAt:     time.Now().AddDate(0, 0, 7).Unix(),
		CreatedAt:    time.Now().Unix(),
	})
	if err != nil {
		return res, errors.New("ManageShareLogic.CreateSessionShareToken.ShareTokenStore.Create", i18n.ERROR_INTERNAL, err)
	}

	return res, nil
}

type ShareLogic struct {
	ctx  context.Context
	core *core.Core
}

func NewShareLogic(ctx context.Context, core *core.Core) *ShareLogic {
	l := &ShareLogic{
		ctx:  ctx,
		core: core,
	}

	return l
}

type KnowledgeShareInfo struct {
	UserID       string                     `json:"user_id"`
	UserName     string                     `json:"user_name"`
	UserAvatar   string                     `json:"user_avatar"`
	KnowledgeID  string                     `json:"knowledge_id"`
	SpaceID      string                     `json:"space_id"`
	Kind         types.KnowledgeKind        `json:"kind" db:"kind"`
	Title        string                     `json:"title" db:"title"`
	Tags         pq.StringArray             `json:"tags" db:"tags"`
	Content      types.KnowledgeContent     `json:"content" db:"content"`
	ContentType  types.KnowledgeContentType `json:"content_type" db:"content_type"`
	CreatedAt    int64                      `json:"created_at"`
	EmbeddingURL string                     `json:"embedding_url"`
}

func (l *ShareLogic) GetShareLink(token string) (*types.ShareToken, error) {
	link, err := l.core.Store().ShareTokenStore().GetByToken(l.ctx, token)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetShareLink.ShareTokenStore.GetByToken", i18n.ERROR_INTERNAL, err)
	}

	if link == nil {
		return nil, errors.New("ShareLogic.GetShareLink.ShareTokenStore.GetByToken.nil", i18n.ERROR_NOT_FOUND, nil).Code(http.StatusNoContent)
	}

	return link, nil
}

func (l *ShareLogic) GetKnowledgeByShareToken(token string) (*KnowledgeShareInfo, error) {
	link, err := l.core.Store().ShareTokenStore().GetByToken(l.ctx, token)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetKnowledgeByShareToken.ShareTokenStore.GetByToken", i18n.ERROR_INTERNAL, err)
	}

	if link == nil {
		return nil, errors.New("ShareLogic.GetKnowledgeByShareToken.ShareTokenStore.GetByToken.nil", i18n.ERROR_NOT_FOUND, nil).Code(http.StatusNoContent)
	}

	knowledge, err := l.core.Store().KnowledgeStore().GetKnowledge(l.ctx, link.SpaceID, link.ObjectID)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetKnowledgeByShareToken.KnowledgeStore.GetKnowledge", i18n.ERROR_INTERNAL, err)
	}

	if knowledge == nil {
		return nil, errors.New("ShareLogic.GetKnowledgeByShareToken.KnowledgeStore.GetKnowledge.nil", i18n.ERROR_NOT_FOUND, nil).Code(http.StatusNoContent)
	}

	user, err := l.core.Store().UserStore().GetUser(l.ctx, link.Appid, knowledge.UserID)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetKnowledgeByShareToken.UserStore.GetUser", i18n.ERROR_INTERNAL, err)
	}

	if user == nil {
		user = &types.User{
			Name: "Null",
			ID:   "",
		}
	}

	return &KnowledgeShareInfo{
		UserID:       user.ID,
		UserName:     user.Name,
		UserAvatar:   user.Avatar,
		KnowledgeID:  knowledge.ID,
		SpaceID:      knowledge.SpaceID,
		Kind:         knowledge.Kind,
		Title:        knowledge.Title,
		Tags:         knowledge.Tags,
		Content:      knowledge.Content,
		ContentType:  knowledge.ContentType,
		CreatedAt:    knowledge.CreatedAt,
		EmbeddingURL: link.EmbeddingURL,
	}, nil
}

type SessionShareInfo struct {
	User         *types.User          `json:"user"`
	Session      *types.ChatSession   `json:"session"`
	Messages     []*types.ChatMessage `json:"messages"`
	EmbeddingURL string               `json:"embedding_url"`
}

func (l *ShareLogic) GetSessionByShareToken(token string) (*SessionShareInfo, error) {
	link, err := l.core.Store().ShareTokenStore().GetByToken(l.ctx, token)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetSessionByShareToken.ShareTokenStore.GetByToken", i18n.ERROR_INTERNAL, err)
	}

	if link == nil {
		return nil, errors.New("ShareLogic.GetSessionByShareToken.ShareTokenStore.GetByToken.nil", i18n.ERROR_NOT_FOUND, nil).Code(http.StatusNoContent)
	}

	session, err := l.core.Store().ChatSessionStore().GetChatSession(l.ctx, link.SpaceID, link.ObjectID)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetSessionByShareToken.ChatSessionStore.GetChatSession", i18n.ERROR_INTERNAL, err)
	}

	if session == nil {
		return nil, errors.New("ShareLogic.GetSessionByShareToken.ChatSessionStore.GetChatSession.nil", i18n.ERROR_NOT_FOUND, nil).Code(http.StatusNoContent)
	}

	messageList, err := l.core.Store().ChatMessageStore().ListSessionMessage(l.ctx, link.SpaceID, link.ObjectID, "", types.NO_PAGING, types.NO_PAGING)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetSessionByShareToken.ChatMessageStore.ListSessionMessage", i18n.ERROR_INTERNAL, err)
	}

	if len(messageList) == 0 {
		return nil, errors.New("ShareLogic.GetSessionByShareToken.ChatMessageStore.ListSessionMessage.nil", i18n.ERROR_NOT_FOUND, nil).Code(http.StatusNoContent)
	}

	user, err := l.core.Store().UserStore().GetUser(l.ctx, link.Appid, link.ShareUserID)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("ShareLogic.GetKnowledgeByShareToken.UserStore.GetUser", i18n.ERROR_INTERNAL, err)
	}

	if user == nil {
		user = &types.User{
			Name:   "Null",
			Avatar: l.core.Cfg().Site.DefaultAvatar,
			ID:     "",
		}
	}

	return &SessionShareInfo{
		User:         user,
		Session:      session,
		Messages:     lo.Reverse(messageList),
		EmbeddingURL: link.EmbeddingURL,
	}, nil
}
