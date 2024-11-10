package deepseek_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/breeew/brew-api/pkg/ai"
	openai "github.com/breeew/brew-api/pkg/ai/deepseek"
	"github.com/breeew/brew-api/pkg/types"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

func new() *openai.Driver {
	return openai.New(os.Getenv("BREW_API_AI_LANG"), os.Getenv("BREW_API_AI_DEEPSEEK_TOKEN"), os.Getenv("BREW_API_AI_DEEPSEEK_ENDPOINT"), ai.ModelName{
		ChatModel: "deepseek-chat",
	})
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
		以下是关于回答用户提问可以参考的内容(json格式)：
		--------------------------------------
		{solt}
		--------------------------------------
		你需要结合“参考内容”来回答用户的提问，如果参考内容完全没有用户想要的结果，请告诉用户你无法回答该内容，因为你并没有为此记录过任何内容。
		注意，“参考内容”中可能有部分内容描述的是同一件事情，但是发生的时间不同，当你无法选择应该参考哪一天的内容时，可以结合用户提出的问题进行分析。
		如果你从上述内容中找到了用户想要的答案，可以结合内容相关的属性来给到用户更多的帮助，比如告诉用户这件事发生在哪天(参考date_time属性)，注意，“参考内容”中提到的时间并不一定是基于现在时间所描述出来的，你需要根据date_time及时间表来给出正确的时间描述。
		请你使用 {lang} 语言，以Markdown格式回复用户。
	`)
	opts.WithDocs([]*types.PassageInfo{
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
