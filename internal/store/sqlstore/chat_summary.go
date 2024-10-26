package sqlstore

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/starbx/brew-api/pkg/register"
	"github.com/starbx/brew-api/pkg/types"
)

func init() {
	register.RegisterFunc(registerKey{}, func() {
		provider.stores.ChatSummaryStore = NewChatSummaryStore(provider)
	})
}

type ChatSummaryStore struct {
	CommonFields
}

func NewChatSummaryStore(provider SqlProviderAchieve) *ChatSummaryStore {
	repo := &ChatSummaryStore{}
	repo.SetProvider(provider)
	repo.SetTable(types.TABLE_CHAT_SUMMARY)
	repo.SetAllColumns("id", "message_id", "session_id", "content", "created_at")
	return repo
}

func (s *ChatSummaryStore) GetChatSessionLatestSummary(ctx context.Context, sessionID string) (*types.ChatSummary, error) {
	query := sq.Select(s.GetAllColumns()...).From(s.GetTable()).Where(sq.Eq{"session_id": sessionID}).OrderBy("created_at DESC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errorSqlBuild(err)
	}

	var res types.ChatSummary
	if err = s.GetReplica(ctx).Get(&res, queryString, args...); err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *ChatSummaryStore) Create(ctx context.Context, data types.ChatSummary) error {
	if data.CreatedAt == 0 {
		data.CreatedAt = time.Now().Unix()
	}
	query := sq.Insert(s.GetTable()).
		Columns("id", "message_id", "session_id", "content", "created_at").
		Values(data.ID, data.MessageID, data.SessionID, data.Content, data.CreatedAt)

	queryString, args, err := query.ToSql()
	if err != nil {
		return errorSqlBuild(err)
	}

	_, err = s.GetMaster(ctx).Exec(queryString, args...)
	if err != nil {
		return err
	}
	return nil
}
