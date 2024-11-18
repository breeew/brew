package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/breeew/brew-api/internal/core"
	v1 "github.com/breeew/brew-api/internal/logic/v1"
	"github.com/breeew/brew-api/pkg/s3"
	"github.com/breeew/brew-api/pkg/safe"
	"github.com/breeew/brew-api/pkg/types"
	"github.com/breeew/brew-api/pkg/utils"
)

func NewSingleLock() *SingleLock {
	return &SingleLock{
		locks: make(map[string]bool),
	}
}

type SelfHostCustomConfig struct {
	S3 S3Config `toml:"s3"`
}

type S3Config struct {
	Bucket    string `toml:"bucket"`
	Region    string `toml:"region"`
	Endpoint  string `toml:"endpoint"`
	AccessKey string `toml:"access_key"`
	SecretKey string `toml:"secret_key"`
}

type SingleLock struct {
	mu    sync.Mutex
	locks map[string]bool
}

func (s *SingleLock) TryLock(ctx context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locks[key] {
		return false, nil
	}
	go safe.Run(func() {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			defer s.mu.Unlock()
			delete(s.locks, key)
		}
	})
	return true, nil
}

var _ core.Plugins = (*SelfHostPlugin)(nil)

func newSelfHostMode() *SelfHostPlugin {
	return &SelfHostPlugin{
		Appid:      "brew-selfhost",
		singleLock: NewSingleLock(),
	}
}

type SelfHostPlugin struct {
	core       *core.Core
	Appid      string
	singleLock *SingleLock

	S3Config *S3Config
}

func (s *SelfHostPlugin) DefaultAppid() string {
	return s.Appid
}

func (s *SelfHostPlugin) Install(c *core.Core) error {
	s.core = c
	fmt.Println("Start initialize.")
	utils.SetupIDWorker(1)

	customConfig := core.NewCustomConfigPayload[SelfHostCustomConfig]()
	if err := s.core.Cfg().LoadCustomConfig(&customConfig); err != nil {
		return fmt.Errorf("Failed to install custom config, %w", err)
	}
	s.S3Config = &customConfig.CustomConfig.S3

	var tokenCount int
	if err := s.core.Store().GetMaster().Get(&tokenCount, "SELECT COUNT(*) FROM "+types.TABLE_ACCESS_TOKEN.Name()+" WHERE true"); err != nil {
		return fmt.Errorf("Initialize sql error: %w", err)
	}

	if tokenCount > 0 {
		fmt.Println("System is already initialized. Skip.")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	userID := utils.GenRandomID()
	var (
		token   string
		spaceID string
		err     error
	)

	err = s.core.Store().Transaction(ctx, func(ctx context.Context) error {
		authLogic := v1.NewAuthLogic(ctx, s.core)
		token, err = authLogic.GenAccessToken(s.Appid, "Initialize the self-host token.", userID, time.Now().AddDate(999, 0, 0).Unix())
		if err != nil {
			return err
		}

		tokenInfo, err := authLogic.GetAccessTokenDetail(s.Appid, token)
		if err != nil {
			return err
		}

		claims, err := tokenInfo.TokenClaims()
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, v1.TOKEN_CONTEXT_KEY, claims)
		spaceID, err = v1.NewSpaceLogic(ctx, s.core).CreateUserSpace("default", "default")
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("Appid:", s.Appid)
	fmt.Println("Access token:", token)
	fmt.Println("Space id:", spaceID)
	return nil
}

func (s *SelfHostPlugin) TryLock(ctx context.Context, key string) (bool, error) {
	return s.singleLock.TryLock(ctx, key)
}

func (s *SelfHostPlugin) AIChatLogic() core.AIChatLogic {
	return v1.NewNormalAssistant(s.core)
}

var limiter = make(map[string]*rate.Limiter)

// ratelimit 代表每分钟允许的数量
func (s *SelfHostPlugin) UseLimiter(key string, method string, defaultRatelimit int) core.Limiter {
	l, exist := limiter[key]
	if !exist {
		limit := rate.Every(time.Minute / time.Duration(defaultRatelimit))
		limiter[key] = rate.NewLimiter(limit, defaultRatelimit*2)
		l = limiter[key]
	}

	return l
}

func (s *SelfHostPlugin) FileUploader() core.FileStorage {
	if s.S3Config != nil {
		return &S3FileStorage{
			S3: s3.NewS3Client(s.S3Config.Endpoint, s.S3Config.Region, s.S3Config.Bucket, s.S3Config.AccessKey, s.S3Config.SecretKey),
		}
	}
	return &LocalFileStorage{}
}
