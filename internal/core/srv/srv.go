package srv

import (
	"github.com/starbx/brew-api/pkg/socket/firetower"
)

type Srv struct {
	rbac  *RBACSrv
	ai    *AI
	tower *Tower
	seq   *SeqSrv
}

func SetupSrvs(opts ...ApplyFunc) *Srv {
	a := &Srv{
		rbac: SetupRBACSrv(), // 角色鉴权
	}

	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (s *Srv) RBAC() *RBACSrv {
	return s.rbac
}

func (s *Srv) AI() AIDriver {
	return s.ai
}

func (s *Srv) SeqSrv() *SeqSrv {
	return s.seq
}

func (t *Tower) Pusher() *firetower.SelfPusher[PublishData] {
	return t.pusher
}

func (s *Srv) Tower() *Tower {
	return s.tower
}
