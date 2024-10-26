package core

import (
	"context"

	"github.com/starbx/brew-api/pkg/ai"
	"github.com/starbx/brew-api/pkg/types"
)

type Plugins interface {
	Install(*Core) error
	DefaultAppid() string
	TryLock(ctx context.Context, key string) (bool, error)
	AIChatLogic() AIChatLogic
	UseLimiter(key string, method string, defaultRatelimit int) Limiter
}

type AIChatLogic interface {
	InitAssistantMessage(ctx context.Context, userMessage *types.ChatMessage, ext types.ChatMessageExt) (*types.ChatMessage, error)
	MargeDocsToUserMessage(tpl string, docs []*ai.PassageInfo, userMessage *types.ChatMessage)
	RequestAssistant(ctx context.Context, reqMsgInfo *types.ChatMessage, recvMsgInfo *types.ChatMessage) error
}

type Limiter interface {
	Allow() bool
}

type SetupFunc func() Plugins

func (c *Core) InstallPlugins(p Plugins) {
	p.Install(c)
	c.Plugins = p
}
