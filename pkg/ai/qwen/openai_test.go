package qwen_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/starbx/brew-api/pkg/ai"
	openai "github.com/starbx/brew-api/pkg/ai/qwen"
	"github.com/starbx/brew-api/pkg/types"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

func new() *openai.Driver {
	fmt.Println(os.Getenv("BREW_API_AI_ALI_TOKEN"), os.Getenv("BREW_API_AI_ALI_ENDPOINT"))
	return openai.New(os.Getenv("BREW_API_AI_ALI_TOKEN"), os.Getenv("BREW_API_AI_ALI_ENDPOINT"), ai.ModelName{
		ChatModel:      "qwen-plus",
		EmbeddingModel: "text-embedding-v3",
	})
}

func Test_Embedding(t *testing.T) {
	d := new()
	f, err := os.Create("./vectors")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	res, err := d.EmbeddingForDocument(ctx, "test", []string{"Docker对centos的支持情况怎么样", "Docker"})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(len(res))

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(raw); err != nil {
		t.Fatal(err)
	}

	assert.Greater(t, len(res), 0)

	t.Log(res)
}

func Test_Generate(t *testing.T) {
	d := new()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	opts := d.NewQuery(ctx, []*types.MessageContext{
		{
			Role:    types.USER_ROLE_USER,
			Content: "我的车现在停在哪里？",
		},
	})
	opts.WithDocsSoltName("{solt}").WithPrompt(`
		以下是关于回答用户提问的“参考内容”，这些内容都是历史记录，其中提到的时间点无法与当前时间进行参照：
		--------------------------------------
		{solt}
		--------------------------------------
		你需要结合“参考内容”来回答用户的提问，
		注意，“参考内容”中可能有部分内容描述的是同一件事情，但是发生的时间不同，当你无法选择应该参考哪一天的内容时，可以结合用户提出的问题进行分析。
		如果你从上述内容中找到了用户想要的答案，可以结合内容相关的属性来给到用户更多的帮助，比如参考“事件发生时间”来告诉用户这件事发生在哪天。
		请你使用 {lang} 语言，以Markdown格式回复用户。
	`)
	opts.WithDocs([]*ai.PassageInfo{
		{
			ID:       "xcjoijoijo12",
			Content:  "我有一辆白色的车",
			DateTime: "2024-06-03 15:20:10",
		},
		{
			ID:       "xcjoiaajoijo12",
			Content:  "我有一辆白色的自行车",
			DateTime: "2024-06-03 15:20:10",
		},
		{
			ID:       "3333oij1111oijo12",
			Content:  "周五我把车停在了B3层",
			DateTime: "2024-09-03 15:20:10",
		},
		{
			ID:       "xcjoij12312ijo12",
			Content:  "我昨天把车停在了B2层",
			DateTime: "2024-09-20 15:20:10",
		},
		{
			ID:       "3333oijoijo12",
			Content:  "停车楼里有十辆车",
			DateTime: "2024-06-03 15:20:10",
		},
	})
	res, err := opts.Query()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res)
}

func Test_Summarize(t *testing.T) {
	content := `
通过docker部署向量数据库postgres，pgvector的docker部署方式：
docker run --restart=always \
-id \
--name=postgresql \
-v postgre-data:/var/lib/postgresql/data \
-p 5432:5432 \
-e POSTGRES_PASSWORD=123456 \
-e LANG=C.UTF-8 \
-e POSTGRES_USER=root \
pgvector/pgvector:pg16
	`

	d := new()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	resp, err := d.Summarize(ctx, &content)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}

func Test_EnhanceQuery(t *testing.T) {
	query := "我昨天干啥了？"

	d := new()
	opts := ai.NewEnhance(context.Background(), d)
	res, err := opts.EnhanceQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}

func Test_Chunk(t *testing.T) {
	content := `通过docker部署向量数据库postgres，pgvector的docker部署方式：
docker run --restart=always \
-id \
--name=postgresql \
-v postgre-data:/var/lib/postgresql/data \
-p 5432:5432 \
-e POSTGRES_PASSWORD=123456 \
-e LANG=C.UTF-8 \
-e POSTGRES_USER=root \
pgvector/pgvector:pg16`
	d := new()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	resp, err := d.Chunk(ctx, &content)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}
