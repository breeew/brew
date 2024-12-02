package v1_test

import (
	"context"
	"os"
	"testing"

	"github.com/breeew/brew-api/app/core"
	v1 "github.com/breeew/brew-api/app/logic/v1"
	"github.com/breeew/brew-api/pkg/plugins"
	"github.com/stretchr/testify/assert"
)

func newCore() *core.Core {
	core := core.MustSetupCore(core.MustLoadBaseConfig(os.Getenv("TEST_CONFIG_PATH")))
	plugins.Setup(core.InstallPlugins, "saas")
	return core
}

func Test_UserRegister(t *testing.T) {
	core := newCore()
	logic := v1.NewUserLogic(context.Background(), core)

	userName := ""
	userEmail := ""

	userID, err := logic.Register(core.DefaultAppid(), userName, userEmail, "testpwd")
	if err != nil {
		t.Fatal(err)
	}

	user, err := logic.GetUser(core.DefaultAppid(), userID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, user.Name, userName)
	t.Log(userID)
}
