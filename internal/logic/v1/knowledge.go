package v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/samber/lo"

	"github.com/starbx/brew-api/internal/core"
	"github.com/starbx/brew-api/internal/core/srv"
	"github.com/starbx/brew-api/internal/logic/v1/process"
	"github.com/starbx/brew-api/pkg/ai"
	"github.com/starbx/brew-api/pkg/errors"
	"github.com/starbx/brew-api/pkg/i18n"
	"github.com/starbx/brew-api/pkg/mark"
	"github.com/starbx/brew-api/pkg/safe"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

const CONTEXT_SCENE_PROMPT = `
以下是关于回答用户提问的“参考内容”，这些内容都是历史记录，其中提到的时间点无法与当前时间进行参照：
--------------------------------------
{solt}
--------------------------------------
你需要结合“参考内容”来回答用户的提问，
注意，“参考内容”中可能有部分内容描述的是同一件事情，但是发生的时间不同，当你无法选择应该参考哪一天的内容时，可以结合用户提出的问题进行分析。
如果你从上述内容中找到了用户想要的答案，可以结合内容相关的属性来给到用户更多的帮助，比如参考“事件发生时间”来告诉用户这件事发生在哪天。
请你使用 {lang} 语言，以Markdown格式回复用户。
`

var (
	userSetting = map[string][]ai.OptionFunc{
		"context": {
			func(opts *ai.QueryOptions) {
				opts.WithPrompt(CONTEXT_SCENE_PROMPT)
				opts.WithDocsSoltName("{solt}")
			},
		},
	}
	// resource setting
	// model,prompt,docs_solt,cycle(int days)
)

type KnowledgeLogic struct {
	UserInfo
	ctx  context.Context
	core *core.Core
}

func NewKnowledgeLogic(ctx context.Context, core *core.Core) *KnowledgeLogic {
	l := &KnowledgeLogic{
		ctx:      ctx,
		core:     core,
		UserInfo: setupUserInfo(ctx, core),
	}

	return l
}

func (l *KnowledgeLogic) GetKnowledge(spaceID, id string) (*types.Knowledge, error) {
	data, err := l.core.Store().KnowledgeStore().GetKnowledge(l.ctx, spaceID, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("KnowledgeLogic.GetKnowledge.KnowledgeStore.GetKnowledge", i18n.ERROR_INTERNAL, err)
	}

	if data == nil {
		return nil, errors.New("KnowledgeLogic.GetKnowledge.KnowledgeStore.GetKnowledge.nil", i18n.ERROR_NOTFOUND, err).Code(http.StatusNotFound)
	}

	return data, nil
}

func (l *KnowledgeLogic) ListKnowledges(spaceID string, keywords string, resource *types.ResourceQuery, page, pagesize uint64) ([]*types.Knowledge, uint64, error) {
	opts := types.GetKnowledgeOptions{
		SpaceID:  spaceID,
		Resource: resource,
		Keywords: keywords,
	}
	list, err := l.core.Store().KnowledgeStore().ListKnowledges(l.ctx, opts, page, pagesize)
	if err != nil && err != sql.ErrNoRows {
		return nil, 0, errors.New("KnowledgeLogic.ListKnowledge.KnowledgeStore.ListKnowledge", i18n.ERROR_INTERNAL, err)
	}

	total, err := l.core.Store().KnowledgeStore().Total(l.ctx, opts)
	if err != nil {
		return nil, 0, errors.New("KnowledgeLogic.ListKnowledge.KnowledgeStore.Total", i18n.ERROR_INTERNAL, err)
	}

	return list, total, nil
}

func (l *KnowledgeLogic) Delete(spaceID, id string) error {
	user := l.GetUserInfo()
	if err := l.core.Srv().RBAC().Check(user, l.lazyRolerFromKnowledgeID(spaceID, id), srv.PermissionEdit); err != nil {
		return errors.Trace("KnowledgeLogic.Delete", err)
	}

	return l.core.Store().Transaction(l.ctx, func(ctx context.Context) error {
		if err := l.core.Store().KnowledgeStore().Delete(ctx, spaceID, id); err != nil {
			return errors.New("KnowledgeLogic.Delete.KnowledgeStore.Delete", i18n.ERROR_INTERNAL, err)
		}

		if err := l.core.Store().KnowledgeChunkStore().BatchDelete(ctx, spaceID, id); err != nil {
			return errors.New("KnowledgeLogic.Delete.KnowledgeChunkStore.BatchDelete", i18n.ERROR_INTERNAL, err)
		}

		if err := l.core.Store().VectorStore().BatchDelete(ctx, spaceID, id); err != nil {
			return errors.New("KnowledgeLogic.Delete.VectorStore.Delete", i18n.ERROR_INTERNAL, err)
		}

		return nil
	})
}

func (l *KnowledgeLogic) Update(spaceID, id string, args types.UpdateKnowledgeArgs) error {
	oldKnowledge, err := l.core.Store().KnowledgeStore().GetKnowledge(l.ctx, spaceID, id)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("KnowledgeLogic.Update.KnowledgeStore.GetKnowledge", i18n.ERROR_INTERNAL, err)
	}

	if oldKnowledge == nil || oldKnowledge.UserID != l.GetUserInfo().User {
		return errors.New("KnowledgeLogic.Update.KnowledgeStore.GetKnowledge", i18n.ERROR_NOTFOUND, err).Code(http.StatusNotFound)
	}

	tagsChanged := false
	if len(args.Tags) != 0 {
		if len(args.Tags) != len(oldKnowledge.Tags) {
			tagsChanged = true
		} else {
			for _, v := range args.Tags {
				matched := false
				for _, vv := range oldKnowledge.Tags {
					if v == vv {
						matched = true
						break
					}
				}
				if !matched {
					tagsChanged = true
					break
				}
			}
		}
	}

	var summary []string
	if !tagsChanged {
		summary = append(summary, "tags")
	}
	if string(args.Content) != string(oldKnowledge.Content) {
		summary = append(summary, "content")
	}
	if args.Title == "" {
		summary = append(summary, "title")
	}

	err = l.core.Store().KnowledgeStore().Update(l.ctx, spaceID, id, types.UpdateKnowledgeArgs{
		Resource: args.Resource,
		Title:    args.Title,
		Content:  args.Content,
		Tags:     args.Tags,
		Stage:    types.KNOWLEDGE_STAGE_SUMMARIZE,
		Kind:     args.Kind,
		Summary:  strings.Join(summary, ","),
	})
	if err != nil {
		return errors.New("KnowledgeLogic.Update.KnowledgeStore.Update", i18n.ERROR_INTERNAL, err)
	}

	go safe.Run(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
		defer cancel()
		knowledge, err := l.core.Store().KnowledgeStore().GetKnowledge(ctx, spaceID, id)
		if err != nil {
			slog.Error("Failed to get new knowledge after update, knowledge process stopped",
				slog.String("space_id", spaceID),
				slog.String("knowledge_id", id),
				slog.String("error", err.Error()))
			return
		}
		if err = l.processKnowledgeAsync(*knowledge); err != nil {
			slog.Error("Process knowledge async failed",
				slog.String("space_id", knowledge.SpaceID),
				slog.String("knowledge_id", knowledge.ID),
				slog.Any("error", err))
		}
	})

	return nil
}

func (l *KnowledgeLogic) GetQueryRelevanceKnowledges(spaceID, userID, query string, resource *types.ResourceQuery) (*types.RAGDocs, error) {
	var result types.RAGDocs
	aiOpts := l.core.Srv().AI().NewEnhance(l.ctx)
	aiOpts.WithPrompt(l.core.Cfg().Prompt.EnhanceQuery)
	resp, err := aiOpts.EnhanceQuery(query)
	if err != nil {
		slog.Error("failed to enhance user query", slog.String("query", query), slog.String("error", err.Error()))
		// return nil, errors.New("KnowledgeLogic.GetRelevanceKnowledges.AI.EnhanceQuery", i18n.ERROR_INTERNAL, err)
	}

	queryStrs := []string{query}
	if len(resp.News) > 0 {
		queryStrs = append(queryStrs, resp.News...)
	}

	vector, err := l.core.Srv().AI().EmbeddingForQuery(l.ctx, []string{strings.Join(queryStrs, " ")})
	if err != nil || len(vector) == 0 {
		return nil, errors.New("KnowledgeLogic.GetRelevanceKnowledges.AI.EmbeddingForQuery", i18n.ERROR_INTERNAL, err)
	}

	refs, err := l.core.Store().VectorStore().Query(l.ctx, types.GetVectorsOptions{
		SpaceID:  spaceID,
		UserID:   userID,
		Resource: resource,
	}, pgvector.NewVector(vector[0]), 40)
	if err != nil {
		return nil, errors.New("KnowledgeLogic.GetRelevanceKnowledges.VectorStore.Query", i18n.ERROR_INTERNAL, err)
	}

	slog.Debug("got query result", slog.String("query", query), slog.Any("result", refs))
	if len(refs) == 0 {
		return nil, nil
	}

	var (
		knowledgeIDs []string
	)
	for i, v := range refs {
		if i > 0 && v.Cos > 0.5 && v.OriginalLength > 200 {
			// TODO：more and more verify best ratio
			continue
		}

		result.Refs = append(result.Refs, v)
	}

	result.Refs = lo.UniqBy(result.Refs, func(item types.QueryResult) string {
		return item.KnowledgeID
	})

	for _, v := range result.Refs {
		knowledgeIDs = append(knowledgeIDs, v.KnowledgeID)
	}

	knowledges, err := l.core.Store().KnowledgeStore().ListKnowledges(l.ctx, types.GetKnowledgeOptions{
		IDs:      knowledgeIDs,
		SpaceID:  spaceID,
		UserID:   userID,
		Resource: resource,
	}, 1, 20)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("KnowledgeLogic.Query.KnowledgeStore.ListKnowledge", i18n.ERROR_INTERNAL, err)
	}
	if len(knowledges) == 0 {
		// return nil, errors.New("KnowledgeLogic.Query.KnowledgeStore.ListKnowledge.nil", i18n.ERROR_LOGIC_VECTOR_DB_NOT_MATCHED_CONTENT_DB, nil)
	}

	slog.Debug("match knowledges", slog.String("query", query), slog.Any("resource", resource), slog.Int("knowledge_length", len(knowledges)))

	for _, v := range knowledges {
		content := string(v.Content)
		if v.ContentType == types.KNOWLEDGE_CONTENT_TYPE_BLOCKS {
			if content, err = utils.ConvertEditorJSBlocksToMarkdown(v.Content); err != nil {
				slog.Error("Failed to convert editor blocks to markdown", slog.String("knowledge_id", v.ID), slog.String("error", err.Error()))
				continue
			}
		}

		sw := mark.NewSensitiveWork()
		result.Docs = append(result.Docs, &types.PassageInfo{
			ID:       v.ID,
			Content:  sw.Do(content),
			DateTime: v.MaybeDate,
			SW:       sw,
		})
	}
	return &result, nil
}

type KnowledgeQueryResult struct {
	Refs    []types.QueryResult `json:"refs"`
	Message string              `json:"message"`
}

func (l *KnowledgeLogic) Query(spaceID string, resource *types.ResourceQuery, query string) (*KnowledgeQueryResult, error) {
	vector, err := l.core.Srv().AI().EmbeddingForQuery(l.ctx, []string{query})
	if err != nil || len(vector) == 0 {
		return nil, errors.New("KnowledgeLogic.Query.AI.EmbeddingForQuery", i18n.ERROR_INTERNAL, err)
	}

	user := l.GetUserInfo()

	refs, err := l.core.Store().VectorStore().Query(l.ctx, types.GetVectorsOptions{
		SpaceID:  spaceID,
		UserID:   user.User,
		Resource: resource,
	}, pgvector.NewVector(vector[0]), 20)
	if err != nil {
		return nil, errors.New("KnowledgeLogic.Query.VectorStore.Query", i18n.ERROR_INTERNAL, err)
	}

	slog.Debug("got query result", slog.String("query", query), slog.Any("result", refs))
	var result = KnowledgeQueryResult{
		Message: "no content matched",
	}
	// TODO switch mode, no refs no gen || no refs ai gen without docs
	// current is no refs no gen
	if len(refs) == 0 {
		return &result, nil
	}

	var (
		knowledgeIDs []string
		hasMatched   bool
	)
	for _, v := range refs {
		if !hasMatched && v.Cos < 0.5 {
			hasMatched = true
		}
		if hasMatched && v.Cos >= 0.5 {
			continue
		}
		knowledgeIDs = append(knowledgeIDs, v.KnowledgeID)
		result.Refs = append(result.Refs, v)
	}

	knowledges, err := l.core.Store().KnowledgeStore().ListKnowledges(l.ctx, types.GetKnowledgeOptions{
		IDs:     knowledgeIDs,
		SpaceID: spaceID,
		UserID:  user.User,
	}, 1, 20)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.New("KnowledgeLogic.Query.KnowledgeStore.ListKnowledge", i18n.ERROR_INTERNAL, err)
	}
	if len(knowledges) == 0 {
		// return nil, errors.New("KnowledgeLogic.Query.KnowledgeStore.ListKnowledge.nil", i18n.ERROR_LOGIC_VECTOR_DB_NOT_MATCHED_CONTENT_DB, nil)
	}

	slog.Debug("match knowledges", slog.String("query", query), slog.Any("resource", resource), slog.Int("knowledge_length", len(knowledges)))

	var (
		docs []*types.PassageInfo
	)
	for _, v := range knowledges {
		content := string(v.Content)
		if v.ContentType == types.KNOWLEDGE_CONTENT_TYPE_BLOCKS {
			if content, err = utils.ConvertEditorJSBlocksToMarkdown(v.Content); err != nil {
				slog.Error("Failed to convert editor blocks to markdown", slog.String("knowledge_id", v.ID), slog.String("error", err.Error()))
				continue
			}
		}
		sw := mark.NewSensitiveWork()
		docs = append(docs, &types.PassageInfo{
			ID:       v.ID,
			Content:  sw.Do(content),
			DateTime: v.MaybeDate,
			SW:       sw,
		})
	}

	message := &types.MessageContext{
		Role:    types.USER_ROLE_USER,
		Content: query,
	}

	// TODO: gen query opts from user setting
	queryOptions := l.core.Srv().AI().NewQuery(l.ctx, []*types.MessageContext{message})
	if resource != nil && len(resource.Include) == 1 {
		// match user resource setting
		for _, apply := range userSetting[resource.Include[0]] {
			apply(queryOptions)
		}
	}

	resp, err := queryOptions.Query()
	if err != nil {
		return nil, errors.New("KnowledgeLogic.Query.queryOptions.Query", i18n.ERROR_INTERNAL, err)
	}

	result.Message = strings.Join(resp.Received, "\n")

	for _, v := range docs {
		result.Message = v.SW.Undo(result.Message)
	}

	return &result, nil
}

func (l *KnowledgeLogic) insertContent(isSync bool, spaceID, resource string, kind types.KnowledgeKind, content json.RawMessage, contentType types.KnowledgeContentType) (string, error) {
	if resource == "" {
		resource = types.DEFAULT_RESOURCE
	}
	knowledgeID := utils.GenRandomID()
	user := l.GetUserInfo()
	knowledge := types.Knowledge{
		ID:          knowledgeID,
		SpaceID:     spaceID,
		UserID:      user.User,
		Resource:    resource,
		Content:     content,
		ContentType: contentType,
		Kind:        kind,
		Stage:       types.KNOWLEDGE_STAGE_SUMMARIZE,
		MaybeDate:   time.Now().Local().Format("2006-01-02 15:04"),
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}
	err := l.core.Store().KnowledgeStore().Create(l.ctx, knowledge)
	if err != nil {
		return "", errors.New("KnowledgeLogic.InsertContent.Store.KnowledgeStore.Create", i18n.ERROR_INTERNAL, err)
	}

	if isSync {
		if err = l.processKnowledgeAsync(knowledge); err != nil {
			return knowledgeID, errors.Trace("KnowledgeLogic.InsertContent", err)
		}
	} else {
		go safe.Run(func() {
			if err = l.processKnowledgeAsync(knowledge); err != nil {
				slog.Error("Process knowledge async failed",
					slog.String("space_id", knowledge.SpaceID),
					slog.String("knowledge_id", knowledge.ID),
					slog.Any("error", err))
			}
		})
	}

	return knowledgeID, nil
}

const (
	InserTypeSync  = true
	InserTypeAsync = false
)

func (l *KnowledgeLogic) InsertContentAsync(spaceID, resource string, kind types.KnowledgeKind, content json.RawMessage, contentType types.KnowledgeContentType) (string, error) {
	return l.insertContent(InserTypeAsync, spaceID, resource, kind, content, contentType)
}

func (l *KnowledgeLogic) InsertContent(spaceID, resource string, kind types.KnowledgeKind, content json.RawMessage, contentType types.KnowledgeContentType) (string, error) {
	return l.insertContent(InserTypeSync, spaceID, resource, kind, content, contentType)
	// sw := mark.NewSensitiveWork()
	// content = sw.Do(content)

	// // flow start
	// summary, err := l.core.Srv().AI().Summarize(l.ctx, &content)
	// if err != nil {
	// 	return knowledgeID, errors.New("KnowledgeLogic.AI.Summarize", i18n.ERROR_INTERNAL, err)
	// }

	// slog.Debug("knowledge summary result", slog.Any("result", summary))

	// if summary.DateTime == "" {
	// 	summary.DateTime = knowledge.MaybeDate
	// }

	// if summary.Summary == "" {
	// 	summary.Summary = content
	// }

	// embeddingContent := summary.Summary
	// summary.Summary = sw.Undo(summary.Summary)
	// summary.Summary = summary.Title + "\n" + summary.Summary

	// if err = l.core.Store().KnowledgeStore().FinishedStageSummarize(l.ctx, spaceID, knowledgeID, summary); err != nil {
	// 	return knowledgeID, errors.New("KnowledgeLogic.KnowledgeStore.FinishedStageSummarize", i18n.ERROR_INTERNAL, err)
	// }

	// vector, err := l.core.Srv().AI().EmbeddingForDocument(l.ctx, "", embeddingContent)
	// if err != nil {
	// 	return knowledgeID, errors.New("KnowledgeLogic.AI.EmbeddingForDocument", i18n.ERROR_INTERNAL, err)
	// }

	// err = l.core.Store().Transaction(l.ctx, func(ctx context.Context) error {
	// 	err := l.core.Store().VectorStore().Create(ctx, types.Vector{
	// 		ID:        knowledgeID,
	// 		SpaceID:   spaceID,
	// 		UserID:    user.User,
	// 		Embedding: pgvector.NewVector(vector),
	// 		Resource:  resource,
	// 	})
	// 	if err != nil {
	// 		return errors.New("KnowledgeLogic.VectorStore.Create", i18n.ERROR_INTERNAL, err)
	// 	}

	// 	if err = l.core.Store().KnowledgeStore().FinishedStageEmbedding(ctx, spaceID, knowledgeID); err != nil {
	// 		return errors.New("KnowledgeLogic.KnowledgeStore.FinishedStageEmbedding", i18n.ERROR_INTERNAL, err)
	// 	}

	// 	return nil
	// })
	// return knowledgeID, err
}

func (l *KnowledgeLogic) processKnowledgeAsync(knowledge types.Knowledge) error {
	ctx, cancel := context.WithTimeout(l.ctx, time.Minute*2)
	defer cancel()
	respChan := process.NewSummaryRequest(knowledge)
	if respChan == nil {
		return errors.New("KnowledgeLogic.processKnowledgeAsync.NewSummaryRequest", i18n.ERROR_INTERNAL, fmt.Errorf("unexpected, process wrong"))
	}
	select {
	case <-ctx.Done():
		return errors.New("KnowledgeLogic.processKnowledgeAsync.Summary.ctx", i18n.ERROR_INTERNAL, ctx.Err())
	case req := <-respChan:
		if req.Err != nil {
			return errors.New("KnowledgeLogic.processKnowledgeAsync.Summary.Result", i18n.ERROR_INTERNAL, req.Err)
		}
	}

	{
		knowledge, err := l.core.Store().KnowledgeStore().GetKnowledge(ctx, knowledge.SpaceID, knowledge.ID)
		if err != nil {
			return errors.New("KnowledgeLogic.processKnowledgeAsync.KnowledgeStore.GetKnowledge", i18n.ERROR_INTERNAL, err)
		}

		ctx, cancel := context.WithTimeout(l.ctx, time.Minute*2)
		defer cancel()
		respChan := process.NewEmbeddingRequest(*knowledge)
		if respChan == nil {
			return errors.New("KnowledgeLogic.processKnowledgeAsync.NewEmbeddingRequest", i18n.ERROR_INTERNAL, fmt.Errorf("unexpected, process wrong"))
		}
		select {
		case <-ctx.Done():
			return errors.New("KnowledgeLogic.processKnowledgeAsync.Embedding.ctx", i18n.ERROR_INTERNAL, ctx.Err())
		case req := <-respChan:
			if req.Err != nil {
				return errors.New("KnowledgeLogic.processKnowledgeAsync.Embedding.Result", i18n.ERROR_INTERNAL, req.Err)
			}
		}
	}

	return nil
}
