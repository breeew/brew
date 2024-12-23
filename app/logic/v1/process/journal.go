package process

import (
	"context"
	"log"
	"time"

	"github.com/breeew/brew-api/app/store"
	"github.com/breeew/brew-api/pkg/register"
)

type JournalProcess struct {
	store store.JournalStore
}

func NewJournalProcess(store store.JournalStore) *JournalProcess {
	return &JournalProcess{store: store}
}

func (p *JournalProcess) ClearOldJournals(ctx context.Context) error {
	// 获取31天前的日期
	date := time.Now().AddDate(0, 0, -31).Format("2006-01-02")

	// 清理31天前的journals
	err := p.store.DeleteByDate(ctx, date)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	register.RegisterFunc(ProcessKey{}, func(provider *Process) {
		provider.Cron().AddFunc("0 4 * * *", func() {
			err := NewJournalProcess(provider.Core().Store().JournalStore()).ClearOldJournals(context.Background())
			if err != nil {
				log.Printf("Failed to clear old journals: %v", err)
			} else {
				log.Println("Successfully cleared old journals")
			}
		})
	})
}
