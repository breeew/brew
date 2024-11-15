package plugins

import (
	"context"
	"path/filepath"
	"time"

	"github.com/breeew/brew-api/internal/core"
	v1 "github.com/breeew/brew-api/internal/logic/v1"
	"github.com/breeew/brew-api/pkg/utils"
	"golang.org/x/time/rate"
)

var _ core.Plugins = (*SaaSPlugin)(nil)

func newSaaSPlugin() *SaaSPlugin {
	return &SaaSPlugin{
		Appid:      "brew",
		singleLock: NewSingleLock(),
	}
}

type SaaSPlugin struct {
	core       *core.Core
	Appid      string
	singleLock *SingleLock
}

func (s *SaaSPlugin) DefaultAppid() string {
	return s.Appid
}

func (s *SaaSPlugin) Install(c *core.Core) error {
	s.core = c
	utils.SetupIDWorker(1) // TODO: Cluster id by redis
	return nil
}

func (s *SaaSPlugin) TryLock(ctx context.Context, key string) (bool, error) {
	// TODO: Redis lock
	return s.singleLock.TryLock(ctx, key)
}

func (s *SaaSPlugin) AIChatLogic() core.AIChatLogic {
	return v1.NewNormalAssistant(s.core)
}

// ratelimit 代表每分钟允许的数量
func (s *SaaSPlugin) UseLimiter(key string, method string, defaultRatelimit int) core.Limiter {
	l, exist := limiter[key]
	if !exist {
		limit := rate.Every(time.Minute / time.Duration(defaultRatelimit))
		limiter[key] = rate.NewLimiter(limit, defaultRatelimit*2)
		l = limiter[key]
	}

	return l
}

func (s *SaaSPlugin) FileUploader() core.FileStorage {
	return &LocalFileStorage{}
}

type S3 struct{}

func (lfs *S3) GenUploadFileMeta(filePath, fileName string) (core.UploadFileMeta, error) {
	return core.UploadFileMeta{
		FullPath: filepath.Join(filePath, fileName),
	}, nil
}

// SaveFile stores a file on the s3 file system.
func (lfs *S3) SaveFile(filePath, fileName string, content []byte) error {
	// TODO
	return nil
}

// DeleteFile deletes a file from the s3 file system using the full file path.
func (lfs *S3) DeleteFile(fullFilePath string) error {
	// TODO
	return nil
}
