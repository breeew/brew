package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/breeew/brew-api/internal/core"
	v1 "github.com/breeew/brew-api/internal/logic/v1"
	"github.com/breeew/brew-api/pkg/safe"
	"github.com/breeew/brew-api/pkg/types"
	"github.com/breeew/brew-api/pkg/utils"
)

func NewSingleLock() *SingleLock {
	return &SingleLock{
		locks: make(map[string]bool),
	}
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
}

func (s *SelfHostPlugin) DefaultAppid() string {
	return s.Appid
}

func (s *SelfHostPlugin) Install(c *core.Core) error {
	s.core = c
	fmt.Println("Start initialize.")
	utils.SetupIDWorker(1)

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
	return &LocalFileStorage{}
}

type LocalFileStorage struct{}

func (lfs *LocalFileStorage) GenUploadFileMeta(filePath, fileName string) (core.UploadFileMeta, error) {
	return core.UploadFileMeta{
		FullPath: filepath.Join(filePath, fileName),
	}, nil
}

// SaveFile stores a file on the local file system.
func (lfs *LocalFileStorage) SaveFile(filePath, fileName string, content []byte) error {
	// Check if the directory exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// If the directory doesn't exist, create it
		err := os.MkdirAll(filePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	} else if err != nil {
		// If there's an error other than "not exist", return it
		return fmt.Errorf("failed to check directory: %v", err)
	}

	// Save the file
	fullPath := filepath.Join(filePath, fileName)
	err = os.WriteFile(fullPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}

// DeleteFile deletes a file from the local file system using the full file path.
func (lfs *LocalFileStorage) DeleteFile(fullFilePath string) error {
	err := os.Remove(fullFilePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}
