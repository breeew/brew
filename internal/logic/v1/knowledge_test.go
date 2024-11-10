package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/breeew/brew-api/internal/core"
	v1 "github.com/breeew/brew-api/internal/logic/v1"
	"github.com/breeew/brew-api/internal/plugins"
	"github.com/breeew/brew-api/pkg/mark"
	"github.com/breeew/brew-api/pkg/security"
	"github.com/breeew/brew-api/pkg/types"
)

var (
	ctx     context.Context
	spaceid = "DSk2AEZWhHgMeVxKmXKKj97Rrm7TNuOJ"
)

func init() {
	ctx = context.WithValue(context.Background(), v1.TOKEN_CONTEXT_KEY, security.TokenClaims{
		User:    "dhPwO4rHJm4sMe5QPEG6cwM9PmUkKobs",
		Appid:   "test",
		AppName: "brew",
	})
}

func setupCore() *core.Core {
	core := core.MustSetupCore(core.LoadBaseConfigFromENV())
	plugins.Setup(core.InstallPlugins, "selfhost")
	return core
}

func setupKnowledgeLogic() *v1.KnowledgeLogic {
	return v1.NewKnowledgeLogic(ctx, setupCore())
}

func TestKnowledgeInsert(t *testing.T) {
	logic := setupKnowledgeLogic()

	content := "Docker 支持 64 位版本 CentOS 7/8，并且要求内核版本不低于 3.10。 CentOS 7 满足最低内核的要求，但由于内核版本比较低，部分功能（如 overlay2 存储层驱动）无法使用，并且部分功能可能不太稳定。"
	id, err := logic.InsertContent(spaceid, types.DEFAULT_RESOURCE, types.KNOWLEDGE_KIND_TEXT, json.RawMessage(content), types.KNOWLEDGE_CONTENT_TYPE_MARKDOWN)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("new doc", id)
}

func TestKnowledgeQuery(t *testing.T) {
	logic := setupKnowledgeLogic()

	res, err := logic.Query(spaceid, nil, "我昨天做了哪些工作")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(res)
}

func TestHidden(t *testing.T) {
	text := "这是一段测试文本，其中包含 #hidden[secret_data] 和 #hidden[another_secret] 两个隐藏内容。"

	s := mark.NewSensitiveWork()

	text = s.Do(text)
	fmt.Println("do:", text)

	text = s.Undo(text)
	fmt.Println("undo:", text)
}
