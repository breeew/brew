package butler_test

import (
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/breeew/brew-api/app/core"
	"github.com/breeew/brew-api/pkg/ai/agents/butler"
	"github.com/breeew/brew-api/pkg/plugins"
)

func newCore() *core.Core {
	core := core.MustSetupCore(core.MustLoadBaseConfig(os.Getenv("TEST_CONFIG_PATH")))
	return core
}

func newBulter() *butler.ButlerAgent {
	core := newCore()
	plugins.Setup(core.InstallPlugins, "selfhost")
	cfg := openai.DefaultConfig(core.Cfg().AI.Agent.Token)
	cfg.BaseURL = core.Cfg().AI.Agent.Endpoint

	cli := openai.NewClientWithConfig(cfg)
	return butler.NewButlerAgent(core, cli, core.Cfg().AI.Agent.Model, core.Cfg().AI.Agent.VlModel)
}

func TestBulter(t *testing.T) {
	b := newBulter()
	if b == nil {
		t.Fatal("failed to create bulter")
	}

	nextMessage, usage, err := b.Query("tester", "我今天买了 小柴胡颗粒，有效期到 2027年1月20日，请帮我记一下", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(nextMessage, usage)
}
