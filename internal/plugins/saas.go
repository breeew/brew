package plugins

import (
	"context"
	"fmt"
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

type SaaSCustomConfig struct {
	ObjectStorage ObjectStorageDriver `toml:"object_storage"`
}

type SaaSPlugin struct {
	core       *core.Core
	Appid      string
	singleLock *SingleLock

	core.FileStorage

	// custom config
	customConfig SaaSCustomConfig
}

func (s *SaaSPlugin) DefaultAppid() string {
	return s.Appid
}

func (s *SaaSPlugin) Install(c *core.Core) error {
	s.core = c
	utils.SetupIDWorker(1) // TODO: Cluster id by redis

	customConfig := core.NewCustomConfigPayload[SelfHostCustomConfig]()
	if err := s.core.Cfg().LoadCustomConfig(&customConfig); err != nil {
		return fmt.Errorf("Failed to install custom config, %w", err)
	}

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
	if s.FileStorage != nil {
		return s.FileStorage
	}

	s.FileStorage = setupObjectStorage(s.customConfig.ObjectStorage)

	return s.FileStorage
}
