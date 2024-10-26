package types

import (
	"encoding/json"

	"github.com/sashabaranov/go-openai"
)

const AssistantFailedMessage = "Sorry, I'm wrong"

type ChatMessagePart struct {
	Type     openai.ChatMessagePartType  `json:"type,omitempty"`
	Text     string                      `json:"text,omitempty"`
	ImageURL *openai.ChatMessageImageURL `json:"image_url,omitempty"`
}

type MessageContext struct {
	Role         MessageUserRole `json:"role"`
	Content      string          `json:"content"`
	MultiContent []ChatMessagePart
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
