package core

import (
	"context"
	"time"

	"github.com/breeew/brew-api/pkg/types"
)

type Plugins interface {
	Name() string
	Install(*Core) error
	DefaultAppid() string
	TryLock(ctx context.Context, key string) (bool, error)
	UseLimiter(key string, method string, defaultRatelimit int) Limiter
	FileUploader() FileStorage
	AIChatLogic() AIChatLogic
	Cache() Cache
}

type Cache interface {
	SetEx(ctx context.Context, key, value string, expiresAt time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}

type AIChatLogic interface {
	InitAssistantMessage(ctx context.Context, userMessage *types.ChatMessage, ext types.ChatMessageExt) (*types.ChatMessage, error)
	RequestAssistant(ctx context.Context, docs *types.RAGDocs, reqMsgInfo *types.ChatMessage, recvMsgInfo *types.ChatMessage) error
	GetChatSessionSeqID(ctx context.Context, spaceID, sessionID string) (int64, error)
	GenMessageID() string
}

type UploadFileMeta struct {
	UploadEndpoint string `json:"endpoint"`
	FullPath       string `json:"full_path"`
	Domain         string `json:"domain"`
	Status         string `json:"status"`
}

// FileStorage interface defines methods for file operations.
type FileStorage interface {
	GetStaticDomain() string
	GenUploadFileMeta(filePath, fileName string, contentLength int64) (UploadFileMeta, error)
	SaveFile(filePath, fileName string, content []byte) error
	DeleteFile(fullFilePath string) error
	GenGetObjectPreSignURL(url string) (string, error)
}

type Limiter interface {
	Allow() bool
}

type SetupFunc func() Plugins

func (c *Core) InstallPlugins(p Plugins) {
	p.Install(c)
	c.Plugins = p
}