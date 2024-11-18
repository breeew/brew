package core

import (
	"context"

	"github.com/breeew/brew-api/pkg/types"
)

type Plugins interface {
	Install(*Core) error
	DefaultAppid() string
	TryLock(ctx context.Context, key string) (bool, error)
	AIChatLogic() AIChatLogic
	UseLimiter(key string, method string, defaultRatelimit int) Limiter
	FileUploader() FileStorage
}

type AIChatLogic interface {
	InitAssistantMessage(ctx context.Context, userMessage *types.ChatMessage, ext types.ChatMessageExt) (*types.ChatMessage, error)
	RequestAssistant(ctx context.Context, docs *types.RAGDocs, reqMsgInfo *types.ChatMessage, recvMsgInfo *types.ChatMessage) error
}

type UploadFileMeta struct {
	Endpoint string `json:"endpoint"`
	FullPath string `json:"full_path"`
}

// FileStorage interface defines methods for file operations.
type FileStorage interface {
	GenUploadFileMeta(filePath, fileName string) (UploadFileMeta, error)
	SaveFile(filePath, fileName string, content []byte) error
	DeleteFile(fullFilePath string) error
}

type Limiter interface {
	Allow() bool
}

type SetupFunc func() Plugins

func (c *Core) InstallPlugins(p Plugins) {
	p.Install(c)
	c.Plugins = p
}
