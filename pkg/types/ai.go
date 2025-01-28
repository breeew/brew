package types

import (
	"encoding/json"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const (
	AGENT_TYPE_NORMAL  = "rag"
	AGENT_TYPE_JOURNAL = "journal"
	AGENT_TYPE_BULTER  = "bulter"
)

var registeredAgents = map[string][]string{
	AGENT_TYPE_NORMAL:  {},
	AGENT_TYPE_JOURNAL: {"journal", "助理"},
	AGENT_TYPE_BULTER:  {"bulter", "管家"},
}

func FilterAgent(userQuery string) string {
	for agentType, keywords := range registeredAgents {
		for _, keyword := range keywords {
			if strings.Contains(userQuery, "@"+keyword) {
				return agentType
			}
		}
	}
	return AGENT_TYPE_NORMAL
}

const AssistantFailedMessage = "Sorry, I'm wrong"

type ChatMessagePart struct {
	Type     openai.ChatMessagePartType  `json:"type,omitempty"`
	Text     string                      `json:"text,omitempty"`
	ImageURL *openai.ChatMessageImageURL `json:"image_url,omitempty"`
}

type MessageContext struct {
	Role         MessageUserRole `json:"role"`
	Content      string          `json:"content"`
	MultiContent []openai.ChatMessagePart
}

type ResponseChoice struct {
	ID           string
	Message      string
	FinishReason string
	Error        error
}

type MessageContent interface {
	Bytes() json.RawMessage
}

type TextMessage struct {
	Text string `json:"text"`
}

func (t *TextMessage) Bytes() json.RawMessage {
	return json.RawMessage(t.Text)
}
