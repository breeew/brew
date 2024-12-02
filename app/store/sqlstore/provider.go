package sqlstore

import (
	"reflect"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"

	"github.com/breeew/brew-api/app/store"
	"github.com/breeew/brew-api/pkg/register"
	"github.com/breeew/brew-api/pkg/sqlstore"
)

func init() {
	sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

var provider = &Provider{
	stores: &Stores{},
}

func GetProvider() *Provider {
	return provider
}

type Provider struct {
	*sqlstore.SqlProvider
	stores *Stores
}

type Stores struct {
	store.KnowledgeStore
	store.KnowledgeChunkStore
	store.VectorStore
	store.AccessTokenStore
	store.UserSpaceStore
	store.SpaceStore
	store.ResourceStore
	store.UserStore
	store.ChatSessionStore
	store.ChatMessageStore
	store.ChatSummaryStore
	store.ChatMessageExtStore
	store.FileManagementStore
	store.AITokenUsageStore
}

func (s *Provider) batchExecStoreFuncs(fname string) {
	val := reflect.ValueOf(s.stores)
	num := val.NumField()
	for i := 0; i < num; i++ {
		val.Field(i).MethodByName(fname).Call([]reflect.Value{})
	}
}

type registerKey struct{}

func MustSetup(m sqlstore.ConnectConfig, s ...sqlstore.ConnectConfig) func() *Provider {

	provider.SqlProvider = sqlstore.MustSetupProvider(m, s...)

	for _, f := range register.ResolveFuncHandlers(registerKey{}) {
		f()
	}

	return func() *Provider {
		return provider
	}
}

// func (p *Provider) Install() error {
// 	for _, tableFile := range []string{
// 		"access_token.sql",
// 		"chat_message_ext.sql",
// 		"chat_message.sql",
// 		"chat_session.sql",
// 		"chat_summary.sql",
// 		"knowledge_chunk.sql",
// 		"knowledge.sql",
// 		"resource.sql",
// 		"space.sql",
// 		"user_space.sql",
// 		"user.sql",
// 		"vectors.sql",
// 	} {
// 		sql, err := CreateTableFiles.ReadFile(tableFile)
// 		if err != nil {
// 			panic(err)
// 		}

// 		if _, err = p.GetMaster().Exec(string(sql)); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func (p *Provider) store() *Stores {
	return p.stores
}

func (p *Provider) KnowledgeStore() store.KnowledgeStore {
	return p.stores.KnowledgeStore
}

func (p *Provider) VectorStore() store.VectorStore {
	return p.stores.VectorStore
}

func (p *Provider) AccessTokenStore() store.AccessTokenStore {
	return p.stores.AccessTokenStore
}

func (p *Provider) UserSpaceStore() store.UserSpaceStore {
	return p.stores.UserSpaceStore
}

func (p *Provider) SpaceStore() store.SpaceStore {
	return p.stores.SpaceStore
}

func (p *Provider) ResourceStore() store.ResourceStore {
	return p.stores.ResourceStore
}

func (p *Provider) UserStore() store.UserStore {
	return p.stores.UserStore
}

func (p *Provider) KnowledgeChunkStore() store.KnowledgeChunkStore {
	return p.stores.KnowledgeChunkStore
}

func (p *Provider) ChatSessionStore() store.ChatSessionStore {
	return p.stores.ChatSessionStore
}

func (p *Provider) ChatMessageStore() store.ChatMessageStore {
	return p.stores.ChatMessageStore
}

func (p *Provider) ChatSummaryStore() store.ChatSummaryStore {
	return p.stores.ChatSummaryStore
}

func (p *Provider) ChatMessageExtStore() store.ChatMessageExtStore {
	return p.stores.ChatMessageExtStore
}

func (p *Provider) FileManagementStore() store.FileManagementStore {
	return p.stores.FileManagementStore
}

func (p *Provider) AITokenUsageStore() store.AITokenUsageStore {
	return p.stores.AITokenUsageStore
}
