package process

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/holdno/firetower/protocol"
	"github.com/pgvector/pgvector-go"

	"github.com/starbx/brew-api/internal/core"
	"github.com/starbx/brew-api/internal/core/srv"
	"github.com/starbx/brew-api/pkg/mark"
	"github.com/starbx/brew-api/pkg/safe"
	"github.com/starbx/brew-api/pkg/types"
	"github.com/starbx/brew-api/pkg/utils"
)

var (
	knowledgeProcess *KnowledgeProcess
)

func StartKnowledgeProcess(core *core.Core, concurrency int) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	knowledgeProcess = &KnowledgeProcess{
		concurrency:   concurrency,
		ctx:           ctx,
		core:          core,
		SummaryChan:   make(chan *SummaryRequest, 100000),
		EmbeddingChan: make(chan *EmbeddingRequest, 100000),
		processingMap: make(map[string]struct{}),
	}

	go safe.Run(knowledgeProcess.Start)
	go safe.Run(func() {
		knowledgeProcess.Flush()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				knowledgeProcess.Flush()
			}
		}
	})
	return cancel
}

func (p *KnowledgeProcess) Flush() {
	ctx, cancel := context.WithTimeout(p.ctx, time.Second*10)
	defer cancel()
	list, err := p.core.Store().KnowledgeStore().ListProcessingKnowledges(ctx, 3, 1, 20)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("Failed to list processing knowledges", slog.String("error", err.Error()))
		return
	}

	if len(list) > 0 {
		slog.Info("KnowledgeProcess flush", slog.Int("length", len(list)))
	}

	for _, v := range list {
		if v.RetryTimes > 3 {
			continue
		}
		switch v.Stage {
		case types.KNOWLEDGE_STAGE_SUMMARIZE:
			NewSummaryRequest(v)
		case types.KNOWLEDGE_STAGE_EMBEDDING:
			NewEmbeddingRequest(v)
		}
	}
}

type KnowledgeProcess struct {
	concurrency   int
	ctx           context.Context
	core          *core.Core
	SummaryChan   chan *SummaryRequest
	EmbeddingChan chan *EmbeddingRequest
	mu            sync.Mutex
	processingMap map[string]struct{}
}

func (p *KnowledgeProcess) Start() {
	for range 10 {
		go safe.Run(func() {
			p.ProcessSummary()
		})
	}
	for range 10 {
		go safe.Run(func() {
			p.ProcessEmbedding()
		})
	}
}

type SummaryRequest struct {
	ctx      context.Context
	data     types.Knowledge
	response chan SummaryResponse
}

type SummaryResponse struct {
	Err error
}

func (p *KnowledgeProcess) CheckProcess(id string, handler func()) {
	p.mu.Lock()
	if _, exist := p.processingMap[id]; exist {
		p.mu.Unlock()
		return
	}
	p.processingMap[id] = struct{}{}
	p.mu.Unlock()

	handler()
	p.mu.Lock()
	delete(p.processingMap, id)
	p.mu.Unlock()
}

func NewSummaryRequest(data types.Knowledge) chan SummaryResponse {
	if knowledgeProcess == nil || knowledgeProcess.ctx.Err() != nil {
		slog.Error("Knowledge Process not working", slog.String("error", knowledgeProcess.ctx.Err().Error()),
			slog.String("space_id", data.SpaceID), slog.String("knowledge_id", data.ID))
		return nil
	}

	resp := make(chan SummaryResponse, 1)
	knowledgeProcess.SummaryChan <- &SummaryRequest{
		ctx:      context.Background(),
		data:     data,
		response: resp,
	}
	return resp
}

func NewEmbeddingRequest(data types.Knowledge) chan EmbeddingResponse {
	if knowledgeProcess == nil || knowledgeProcess.ctx.Err() != nil {
		slog.Error("Knowledge Process not working", slog.String("error", knowledgeProcess.ctx.Err().Error()),
			slog.String("space_id", data.SpaceID), slog.String("knowledge_id", data.ID))
		return nil
	}

	resp := make(chan EmbeddingResponse, 1)
	knowledgeProcess.EmbeddingChan <- &EmbeddingRequest{
		ctx:      context.Background(),
		data:     data,
		response: resp,
	}
	return resp
}

func (p *KnowledgeProcess) ProcessSummary() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case req := <-p.SummaryChan:
			if req == nil {
				continue
			}

			p.CheckProcess(req.data.ID, func() {
				p.processSummary(req)
			})
		}
	}
}

func (p *KnowledgeProcess) ProcessEmbedding() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case req := <-p.EmbeddingChan:
			if req == nil {
				continue
			}

			p.CheckProcess(req.data.ID, func() {
				p.processEmbedding(req)
			})
		}
	}
}

type EmbeddingRequest struct {
	ctx      context.Context
	data     types.Knowledge
	response chan EmbeddingResponse
}

type EmbeddingResponse struct {
	Err error
}

func (p *KnowledgeProcess) processEmbedding(req *EmbeddingRequest) {
	logAttrs := []any{
		slog.String("space_id", req.data.SpaceID),
		slog.String("knowledge_id", req.data.ID),
		slog.String("component", "KnowledgeProcess.processEmbedding"),
	}

	slog.Info("Receive new embedding request",
		logAttrs...)

	var err error
	defer func() {
		slog.Info("Embedding finished",
			logAttrs...)
		if req.response != nil {
			req.response <- EmbeddingResponse{
				Err: err,
			}
			close(req.response)
		}
		if err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if err = p.core.Store().KnowledgeStore().SetRetryTimes(ctx, req.data.SpaceID, req.data.ID, req.data.RetryTimes+1); err != nil {
				slog.Error("Failed to set knowledge process retry times", slog.String("stage", types.KNOWLEDGE_STAGE_EMBEDDING.String()),
					slog.String("error", err.Error()),
					slog.String("space_id", req.data.SpaceID),
					slog.String("knowledge_id", req.data.ID),
					slog.String("component", "KnowledgeProcess.processEmbedding"))
			}
		}
	}()

	// if req.data.Summary == "" {
	// 	err = errors.New("empty summary")
	// 	return
	// }

	sw := mark.NewSensitiveWork()
	// content := sw.Do(req.data.Summary)

	ctx, cancel := context.WithTimeout(req.ctx, time.Minute*5)
	defer cancel()
	chunksData, err := p.core.Store().KnowledgeChunkStore().List(ctx, req.data.SpaceID, req.data.ID)
	if err != nil {
		slog.Error("Failed to list knowledge chuns", append(logAttrs, slog.String("error", err.Error()))...)
		return
	}

	var (
		vectors []types.Vector
		chunks  []string
	)
	for _, v := range chunksData {
		chunks = append(chunks, sw.Do(v.Chunk))

		vectors = append(vectors, types.Vector{
			ID:             v.ID,
			KnowledgeID:    v.KnowledgeID,
			SpaceID:        v.SpaceID,
			UserID:         v.UserID,
			Resource:       req.data.Resource,
			OriginalLength: v.OriginalLength,
			// Embedding:   pgvector.NewVector(vector),
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		})
	}

	vectorResults, err := p.core.Srv().AI().EmbeddingForDocument(ctx, "", chunks)
	if err != nil {
		slog.Error("Failed to embedding for document", append(logAttrs, slog.String("error", err.Error()))...)
		return
	}

	if len(vectorResults) != len(vectors) {
		slog.Error("Embedding results count not matched chunks count", append(logAttrs, slog.String("error", err.Error()))...)
		return
	}

	for i, v := range vectorResults {
		vectors[i].Embedding = pgvector.NewVector(v)
	}

	err = p.core.Store().Transaction(req.ctx, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(req.ctx, time.Minute)
		defer cancel()

		// exist, err := p.core.Store().VectorStore().GetVector(ctx, req.data.SpaceID, req.data.ID)
		// if err != nil && err != sql.ErrNoRows {
		// 	slog.Error("Failed to check the existence of knowledge", append(logAttrs, slog.String("error", err.Error()))...)
		// 	return err
		// }

		// if exist == nil {
		// 	err = p.core.Store().VectorStore().Create(ctx, types.Vector{
		// 		ID:        req.data.ID,
		// 		SpaceID:   req.data.SpaceID,
		// 		UserID:    req.data.UserID,
		// 		Embedding: pgvector.NewVector(vector),
		// 		Resource:  req.data.Resource,
		// 	})
		// 	if err != nil {
		// 		slog.Error("Failed to insert vector data into vector store", append(logAttrs, slog.String("error", err.Error()))...)
		// 		return err
		// 	}
		// } else {
		// 	err = p.core.Store().VectorStore().Update(ctx, req.data.SpaceID, req.data.ID, pgvector.NewVector(vector))
		// 	if err != nil {
		// 		slog.Error("Failed to update vector data", append(logAttrs, slog.String("error", err.Error()))...)
		// 		return err
		// 	}
		// }

		err := p.core.Store().VectorStore().BatchDelete(ctx, req.data.SpaceID, req.data.ID)
		if err != nil && err != sql.ErrNoRows {
			slog.Error("Failed to check the existence of knowledge", append(logAttrs, slog.String("error", err.Error()))...)
			return err
		}

		err = p.core.Store().VectorStore().BatchCreate(ctx, vectors)
		if err != nil {
			slog.Error("Failed to insert vector data into vector store", append(logAttrs, slog.String("error", err.Error()))...)
			return err
		}

		if err = p.core.Store().KnowledgeStore().FinishedStageEmbedding(ctx, req.data.SpaceID, req.data.ID); err != nil {
			slog.Error("Failed to set knowledge finished embedding stage", append(logAttrs, slog.String("error", err.Error()))...)
			return err
		}

		publishStageChangedMessage(p.core.Srv().Tower(), req.data.SpaceID, req.data.ID, types.KNOWLEDGE_STAGE_DONE)
		return nil
	})
}

func (p *KnowledgeProcess) processSummary(req *SummaryRequest) {
	logAttrs := []any{
		slog.String("space_id", req.data.SpaceID),
		slog.String("knowledge_id", req.data.ID),
		slog.String("component", "KnowledgeProcess.processSummary"),
	}

	slog.Info("Receive new summary request",
		logAttrs...)

	sw := mark.NewSensitiveWork()
	content := sw.Do(req.data.Content)

	var err error
	defer func() {
		slog.Info("Summary finished",
			logAttrs...)
		if req.response != nil {
			req.response <- SummaryResponse{
				Err: err,
			}
			close(req.response)
		}
		if err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if err = p.core.Store().KnowledgeStore().SetRetryTimes(ctx, req.data.SpaceID, req.data.ID, req.data.RetryTimes+1); err != nil {
				slog.Error("Failed to set knowledge process retry times",
					append(logAttrs,
						slog.String("stage", types.KNOWLEDGE_STAGE_SUMMARIZE.String()),
						slog.String("error", err.Error()))...)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(req.ctx, time.Minute*5)
	defer cancel()
	summary, err := p.core.Srv().AI().Chunk(ctx, &content)
	if err != nil {
		slog.Error("Failed to summarize knowledge", append(logAttrs, slog.String("error", err.Error()))...)
		return
	}

	slog.Debug("Knowledge summary result", slog.String("knowledge_id", req.data.ID), slog.String("space_id", req.data.SpaceID), slog.Any("result", summary))

	if summary.DateTime == "" {
		summary.DateTime = req.data.MaybeDate
	}

	if len(summary.Chunks) == 0 {
		summary.Chunks = append(summary.Chunks, req.data.Content)
		// summary.Summary = req.data.Content
	} else {
		// summary.Summary = sw.Undo(summary.Summary)
	}

	var chunks []types.KnowledgeChunk
	for _, v := range summary.Chunks {
		chunks = append(chunks, types.KnowledgeChunk{
			ID:             utils.GenRandomID(),
			SpaceID:        req.data.SpaceID,
			KnowledgeID:    req.data.ID,
			UserID:         req.data.UserID,
			Chunk:          sw.Undo(v),
			OriginalLength: len([]rune(req.data.Content)),
			UpdatedAt:      time.Now().Unix(),
			CreatedAt:      time.Now().Unix(),
		})
	}

	if req.data.Summary != "" {
		needToUpdate := make(map[string]bool)
		for _, v := range strings.Split(req.data.Summary, ",") {
			needToUpdate[v] = true
		}

		if !needToUpdate["title"] {
			summary.Title = ""
		}
		if !needToUpdate["tags"] {
			summary.Tags = nil
		}
		if !needToUpdate["content"] {
			chunks = nil
		}
	}

	// if summary.Title != "" {
	// 	summary.Summary = fmt.Sprintf("%s\n%s\n%s", summary.Title, strings.Join(summary.Tags, ","), summary.Summary)
	// }

	p.core.Store().Transaction(req.ctx, func(ctx context.Context) error {
		if len(chunks) > 0 {
			if err = p.core.Store().KnowledgeChunkStore().BatchDelete(req.ctx, req.data.SpaceID, req.data.ID); err != nil {
				slog.Error("Failed to pre-delete knowledge chunks", append(logAttrs, slog.String("error", err.Error()))...)
				return err
			}

			if err = p.core.Store().KnowledgeChunkStore().BatchCreate(req.ctx, chunks); err != nil {
				slog.Error("Failed to create knowledge chunks", append(logAttrs, slog.String("error", err.Error()))...)
				return err
			}
		}

		if err = p.core.Store().KnowledgeStore().FinishedStageSummarize(req.ctx, req.data.SpaceID, req.data.ID, summary); err != nil {
			slog.Error("Failed to set finished summary stage", append(logAttrs, slog.String("error", err.Error()))...)
			return err
		}

		publishStageChangedMessage(p.core.Srv().Tower(), req.data.SpaceID, req.data.ID, types.KNOWLEDGE_STAGE_EMBEDDING)
		return nil
	})
}

func publishStageChangedMessage(tower *srv.Tower, spaceID, knowledgeID string, stage types.KnowledgeStage) {
	fire := tower.NewFire(protocol.SourceSystem, tower.Pusher())
	fire.Message = protocol.TopicMessage[srv.PublishData]{
		Topic: "/knowledge/list/" + spaceID,
		Type:  protocol.PublishOperation,
		Data: srv.PublishData{
			Version: "v1",
			Subject: "stage_changed",
			Data: map[string]string{
				"knowledge_id": knowledgeID,
				"stage":        stage.String(),
			},
		},
	}

	tower.Publish(fire)
}
