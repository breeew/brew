package v1

import (
	"context"
	"net/http"

	"github.com/breeew/brew-api/internal/core"
	"github.com/breeew/brew-api/internal/core/srv"
	"github.com/breeew/brew-api/pkg/ai"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
)

type ReaderLogic struct {
	ctx  context.Context
	core *core.Core
}

func NewReaderLogic(ctx context.Context, core *core.Core) *ReaderLogic {
	l := &ReaderLogic{
		ctx:  ctx,
		core: core,
	}

	return l
}

func (l *ReaderLogic) Reader(endpoint string) (*ai.ReaderResult, error) {
	res, err := l.core.Srv().AI().Reader(l.ctx, endpoint)
	if err != nil {
		errMsg := i18n.ERROR_INTERNAL
		code := http.StatusInternalServerError

		if err == srv.ERROR_UNSUPPORTED_FEATURE {
			errMsg = i18n.ERROR_UNSUPPORTED_FEATURE
			code = http.StatusForbidden
		}
		return nil, errors.New("ReaderLogic.Reader.Srv.AI.Reader", errMsg, err).Code(code)
	}

	return res, nil
}
