package deepseek

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
	NAME = "DeepSeek"
)

type Driver struct {
	lang   string
	client *openai.Client
	model  ai.ModelName
}

func New(lang, token, proxy string, model ai.ModelName) *Driver {
	cfg := openai.DefaultConfig(token)
	if proxy != "" {
		cfg.BaseURL = proxy
	}

	if model.ChatModel == "" {
		model.ChatModel = "deepseek-chat"
	}
	if model.EmbeddingModel == "" {
		// not support
		model.EmbeddingModel = string("deepseek-chat")
	}

	return &Driver{
		lang:   lang,
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
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
				Description: "你从用户描述内容中分析出对应关键内容或关键技术的标签，以便用户后续归类相关的内容，需要以数组的形式组织该字段的值",
				Items: &jsonschema.Definition{
					Type: jsonschema.String,
				},
			},
			"title": {
				Type:        jsonschema.String,
				Description: "为用户提供的内容自动生成标题填入该字段",
			},
			"summary": {
				Type:        jsonschema.String,
				Description: "请将处理后的总结内容填入该字段中",
			},
			"date_time": {
				Type:        jsonschema.String,
				Description: "用户内容中提到的时间，时间格式为 year-month-day hour:minute，如果无法提取时间，请留空",
			},
		},
		Required: []string{"tags", "title", "summary"},
	}

	f := openai.FunctionDefinition{
		Name:        SummarizeFuncName,
		Description: "对文本内容的预处理结果",
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
			Model:    "deepseek-chat",
			Messages: dialogue,
			Tools:    []openai.Tool{t},
		},
	)
	if err != nil || len(resp.Choices) != 1 {
		return result, fmt.Errorf("Completion error: err:%v len(choices):%v\n", err,
			len(resp.Choices))
	}

	fmt.Println(resp.Choices[0].Message.Content)
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
