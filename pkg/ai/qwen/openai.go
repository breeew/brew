package qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/samber/lo"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/starbx/brew-api/pkg/ai"
	"github.com/starbx/brew-api/pkg/types"
)

const (
	NAME = "qwen"
)

type Driver struct {
	client *openai.Client
	model  ai.ModelName
}

func New(token, proxy string, model ai.ModelName) *Driver {
	cfg := openai.DefaultConfig(token)
	if proxy != "" {
		cfg.BaseURL = proxy
	}

	if model.ChatModel == "" {
		model.ChatModel = "qwen-plus"
	}
	if model.EmbeddingModel == "" {
		model.EmbeddingModel = "text-embedding-v3"
	}

	return &Driver{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

func (s *Driver) embedding(ctx context.Context, title string, content []string) ([][]float32, error) {
	slog.Debug("Embedding", slog.String("driver", NAME))
	queryReq := openai.EmbeddingRequest{
		Model:      openai.EmbeddingModel(s.model.EmbeddingModel),
		Dimensions: 1024,
	}

	var (
		groups   [][]string
		result   [][]float32
		batchMax = 6
	)

	for i, v := range content {
		if i%batchMax == 0 {
			groups = append(groups, []string{})
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], v)
	}

	for _, v := range groups {
		// Create an embedding for the user query
		queryReq.Input = v
		resp, err := s.client.CreateEmbeddings(ctx, queryReq)
		if err != nil {
			return nil, fmt.Errorf("Error creating embedding: %w", err)
		}
		for _, v := range resp.Data {
			result = append(result, v.Embedding)
		}
	}

	return result, nil
}

func (s *Driver) EmbeddingForQuery(ctx context.Context, content []string) ([][]float32, error) {
	return s.embedding(ctx, "", content)
}

func (s *Driver) EmbeddingForDocument(ctx context.Context, title string, content []string) ([][]float32, error) {
	return s.embedding(ctx, title, content)
}

func convertPassageToPrompt(docs []*ai.PassageInfo) string {
	raw, _ := json.MarshalIndent(docs, "", "  ")
	b := strings.Builder{}
	b.WriteString("``` json\n")
	b.Write(raw)
	b.WriteString("\n")
	b.WriteString("```\n")
	return b.String()
}

func (s *Driver) NewQuery(ctx context.Context, query []*types.MessageContext) *ai.QueryOptions {
	opts := ai.NewQueryOptions(ctx, s, query)
	return opts
}

func (s *Driver) NewEnhance(ctx context.Context) *ai.EnhanceOptions {
	return ai.NewEnhance(ctx, s)
}

func (s *Driver) QueryStream(ctx context.Context, query []*types.MessageContext) (*openai.ChatCompletionStream, error) {
	messages := lo.Map(query, func(item *types.MessageContext, _ int) openai.ChatCompletionMessage {
		return openai.ChatCompletionMessage{
			Role:    item.Role.String(),
			Content: item.Content,
		}
	})

	req := openai.ChatCompletionRequest{
		Model:    s.model.ChatModel,
		Messages: messages,
		Stream:   true,
	}

	for _, v := range query {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    v.Role.String(),
			Content: v.Content,
		})
	}

	resp, err := s.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Completion error: %w", err)
	}

	slog.Debug("Query", slog.Any("query_stream", req), slog.String("driver", NAME))

	return resp, nil
}

func (s *Driver) Query(ctx context.Context, query []*types.MessageContext) (ai.GenerateResponse, error) {
	messages := lo.Map(query, func(item *types.MessageContext, _ int) openai.ChatCompletionMessage {
		return openai.ChatCompletionMessage{
			Role:    item.Role.String(),
			Content: item.Content,
		}
	})

	req := openai.ChatCompletionRequest{
		Model:    s.model.ChatModel,
		Messages: messages,
	}

	for _, v := range query {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    v.Role.String(),
			Content: v.Content,
		})
	}

	var result ai.GenerateResponse
	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return result, fmt.Errorf("Completion error: %w", err)

	}

	slog.Debug("Query", slog.Any("query", req), slog.String("driver", NAME))

	result.Received = append(result.Received, resp.Choices[0].Message.Content)
	result.TokenCount = int32(resp.Usage.TotalTokens)

	return result, nil
}

const SummarizeFuncName = "summarize"

func (s *Driver) Summarize(ctx context.Context, doc *string) (ai.SummarizeResult, error) {
	slog.Debug("Summarize", slog.String("driver", NAME))
	// describe the function & its inputs
	params := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"tags": {
				Type:        jsonschema.Array,
				Description: "You analyze the user's description to extract corresponding tags for key content or technologies, allowing the user to categorize related content later. The values for this field should be organized as an array.",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"title": {
				Type:        jsonschema.String,
				Description: "Automatically generate a title for the content provided by the user and fill it into this field.",
			},
			"summary": {
				Type:        jsonschema.String,
				Description: "Please fill this field with the processed summary content.",
			},
			"date_time": {
				Type:        jsonschema.String,
				Description: "The time mentioned in the user's content, formatted as year-month-day hour:minute. If the time cannot be extracted, please leave it empty.",
			},
		},
		Required: []string{"tags", "title", "summary"},
	}

	f := openai.FunctionDefinition{
		Name:        SummarizeFuncName,
		Description: "The result of the pre-processing of the text content.",
		Parameters:  params,
	}
	t := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f,
	}

	// simulate user asking a question that requires the function
	dialogue := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: ai.ReplaceVar(ai.PROMPT_PROCESS_CONTENT_CN)},
		{Role: openai.ChatMessageRoleUser, Content: *doc},
	}
	var result ai.SummarizeResult
	resp, err := s.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    s.model.ChatModel,
			Messages: dialogue,
			Tools:    []openai.Tool{t},
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		return result, fmt.Errorf("Completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
	}
	for _, v := range resp.Choices[0].Message.ToolCalls {
		if v.Function.Name != SummarizeFuncName {
			continue
		}
		if err = json.Unmarshal([]byte(v.Function.Arguments), &result); err != nil {
			return result, fmt.Errorf("failed to unmarshal func call arguments of SummarizeResult, %w", err)
		}
	}

	result.Token = resp.Usage.TotalTokens
	return result, nil
}

func (s *Driver) MsgIsOverLimit(msgs []*types.MessageContext) bool {
	return false
}

type EnhanceQueryResult struct {
	Querys []string `json:"querys"`
}

func (s *Driver) EnhanceQuery(ctx context.Context, prompt, query string) (ai.EnhanceQueryResult, error) {
	slog.Debug("EnhanceQuery", slog.String("driver", NAME))
	// describe the function & its inputs
	params := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"querys": {
				Type:        jsonschema.Array,
				Description: "List possible user query questions in this field.",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
		},
		Required: []string{"querys"},
	}

	f := openai.FunctionDefinition{
		Name:        "enhance_query",
		Description: "Enhance user query information to gather more variations of similar question phrasing.",
		Parameters:  params,
	}
	t := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f,
	}

	req := openai.ChatCompletionRequest{
		Model: s.model.ChatModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    types.USER_ROLE_SYSTEM.String(),
				Content: prompt,
			},
			{
				Role:    types.USER_ROLE_USER.String(),
				Content: query,
			},
		},
		Tools:     []openai.Tool{t},
		MaxTokens: 200,
	}

	var (
		funcCallResult EnhanceQueryResult
		result         ai.EnhanceQueryResult
	)
	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) != 1 {
		return result, fmt.Errorf("Completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
	}

	for _, v := range resp.Choices[0].Message.ToolCalls {
		if v.Function.Name != "enhance_query" {
			continue
		}
		if err = json.Unmarshal([]byte(v.Function.Arguments), &funcCallResult); err != nil {
			return result, fmt.Errorf("failed to unmarshal func call arguments of ChunkResult, %w", err)
		}

		result.News = funcCallResult.Querys
	}

	result.Original = query
	return result, nil
}

func (s *Driver) Chunk(ctx context.Context, doc *string) (ai.ChunkResult, error) {
	slog.Debug("Summarize", slog.String("driver", NAME))
	// describe the function & its inputs
	params := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"tags": {
				Type:        jsonschema.Array,
				Description: "Analyze the user's description to identify tags of relevant key content or technologies, allowing the user to categorize related content later. The values for this field should be organized as an array.",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"title": {
				Type:        jsonschema.String,
				Description: "Automatically generate a title for the content provided by the user and fill it into this field.",
			},
			"chunks": {
				Type:        jsonschema.Array,
				Description: "Place the categorized content blocks into this field.",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"date_time": {
				Type:        jsonschema.String,
				Description: "Analyze the time mentioned in the user's content, formatted as year-month-day hour. If you believe there is no time description provided by the user, please leave it empty.",
			},
		},
		Required: []string{"tags", "title", "chunks"},
	}

	f := openai.FunctionDefinition{
		Name:        "chunk",
		Description: "The result of chunking the text content.",
		Parameters:  params,
	}
	t := openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &f,
	}
	// simulate user asking a question that requires the function
	dialogue := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: ai.ReplaceVar(ai.PROMPT_CHUNK_CONTENT_EN)},
		{Role: openai.ChatMessageRoleUser, Content: strings.ReplaceAll(*doc, "\n", "")},
	}
	var result ai.ChunkResult
	resp, err := s.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    s.model.ChatModel,
			Messages: dialogue,
			Tools:    []openai.Tool{t},
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		return result, fmt.Errorf("Completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
	}

	for _, v := range resp.Choices[0].Message.ToolCalls {
		if v.Function.Name != "chunk" {
			continue
		}
		if err = json.Unmarshal([]byte(v.Function.Arguments), &result); err != nil {
			return result, fmt.Errorf("failed to unmarshal func call arguments of ChunkResult, %w", err)
		}
	}

	result.Token = resp.Usage.TotalTokens
	return result, nil
}
