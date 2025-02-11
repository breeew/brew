package journal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/breeew/brew-api/app/core"
	"github.com/breeew/brew-api/pkg/types"
	"github.com/breeew/brew-api/pkg/utils"
	"github.com/samber/lo"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type JournalAgent struct {
	core   *core.Core
	client *openai.Client
	Model  string
}

func NewJournalAgent(core *core.Core, client *openai.Client, model string) *JournalAgent {
	return &JournalAgent{core: core, client: client, Model: model}
}

var FunctionDefine = lo.Map([]*openai.FunctionDefinition{
	{
		Name:        "searchJournal",
		Description: "查询用户时间范围内的日记",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"startDate": {
					Type:        jsonschema.String,
					Description: "获取用户日记的开始日期，格式为 yyyy-mm-dd",
				},
				"endDate": {
					Type:        jsonschema.String,
					Description: "获取用户日记的截至日期，格式为 yyyy-mm-dd",
				},
			},
			Required: []string{"tableName", "data", "tableDesc"},
		},
	},
}, func(item *openai.FunctionDefinition, _ int) openai.Tool {
	return openai.Tool{
		Function: item,
	}
})

func (b *JournalAgent) Query(spaceID, userID, startDate, endDate string) ([]types.Journal, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	journals, err := b.core.Store().JournalStore().ListWithDate(ctx, spaceID, userID, startDate, endDate)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return journals, nil
}

func (b *JournalAgent) HandleUserRequest(spaceID, userID string, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, *openai.Usage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	resp, err := b.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    b.Model,
			Messages: messages,
			Tools:    FunctionDefine,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to request ai: %w", err)
	}

	// 解析OpenAI的响应
	message := resp.Choices[0].Message
	if message.ToolCalls != nil {
		for _, v := range message.ToolCalls {
			switch v.Function.Name {
			case "searchJournal":
				var params struct {
					StartDate string `json:"startDate"`
					EndDate   string `json:"endDate"`
				}
				if err = json.Unmarshal([]byte(v.Function.Arguments), &params); err != nil {
					return nil, nil, err
				}
				res, err := b.Query(spaceID, userID, params.StartDate, params.EndDate)
				if err != nil {
					return nil, nil, err
				}

				sb := strings.Builder{}

				if len(res) == 0 {
					sb.WriteString("用户在这段时间内没有任何日记")
				} else {
					sb.WriteString("以下是查询到的用户日记信息，格式为：\n------  \n{Date}  \n{Journal Content}  \n------\n")

					for _, v := range res {
						content, err := b.core.DecryptData(v.Content)
						if err != nil {
							return nil, nil, err
						}
						md, err := utils.ConvertEditorJSBlocksToMarkdown(content)
						if err != nil {
							return nil, nil, err
						}
						sb.WriteString(v.Date)
						sb.WriteString("  \n")
						sb.WriteString(md)
						sb.WriteString("  \n------  \n")
					}
				}
messages = append(messages, openai.ChatCompletionMessage{
	Role: ,
})
			default:

			}
		}
	}
	return nil, nil, fmt.Errorf("Unknown function call.")
}
