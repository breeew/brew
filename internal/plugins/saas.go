package plugins

import (
	"context"
	"time"

	"github.com/starbx/brew-api/internal/core"
	v1 "github.com/starbx/brew-api/internal/logic/v1"
	"github.com/starbx/brew-api/pkg/utils"
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
func (s *SaaSPlugin) UseLimiter(key string, method string, defaultRatelimit int)  core.Limiter {
	l, exist := limiter[key]
	if !exist {
		limit := rate.Every(time.Minute / time.Duration(defaultRatelimit))
		limiter[key] = rate.NewLimiter(limit, defaultRatelimit*2)
		l = limiter[key]
	}

	return l
}
