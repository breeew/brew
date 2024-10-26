package srv

import (
	"context"

	"github.com/go-redis/redis/v9"

	"github.com/starbx/brew-api/pkg/utils"
)

type SeqGen interface {
	GetChatMessageSequence(ctx context.Context, sessionID string) (int64, error)
}

type SeqSrv struct {
	redis *redis.Client
	gen   SeqGen
}

// TODO: setup with redis
func SetupSeqSrv(gen SeqGen) *SeqSrv {
	return &SeqSrv{
		gen: gen,
	}
}

func (s *SeqSrv) GenMessageID() string {
	return utils.GenSpecIDStr()
}

func (s *SeqSrv) GetChatSessionSeqID(ctx context.Context, sessionID string) (int64, error) {
	return s.gen.GetChatMessageSequence(ctx, sessionID)
	// key := fmt.Sprintf("seq_srv_%d", dialogID)

	// res, err := s.redis.Incr(ctx, key).Result()
	// if err != nil {
	// 	return 0, errors.New("SeqSrv.GetDialogSeqID.initFunc", i18n.ERROR_INTERNAL, err)
	// }

	// return res, nil
}
