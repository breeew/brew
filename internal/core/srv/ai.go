package srv

import (
	"context"
	"os"
	"strings"

	"github.com/starbx/brew-api/pkg/ai"
	"github.com/starbx/brew-api/pkg/ai/azure_openai"
	"github.com/starbx/brew-api/pkg/ai/openai"
	"github.com/starbx/brew-api/pkg/ai/qwen"
	"github.com/starbx/brew-api/pkg/types"
)

type ChatAI interface {
	Summarize(ctx context.Context, doc *string) (ai.SummarizeResult, error)
	Chunk(ctx context.Context, doc *string) (ai.ChunkResult, error)
	MsgIsOverLimit(msgs []*types.MessageContext) bool
	NewQuery(ctx context.Context, msgs []*types.MessageContext) *ai.QueryOptions
	Lang() string
}

type EnhanceAI interface {
	NewEnhance(ctx context.Context) *ai.EnhanceOptions
}

type EmbeddingAI interface {
	EmbeddingForQuery(ctx context.Context, content []string) ([][]float32, error)
	EmbeddingForDocument(ctx context.Context, title string, content []string) ([][]float32, error)
}

type AIDriver interface {
	EmbeddingAI
	EnhanceAI
	ChatAI
}

type AIConfig struct {
	Gemini Gemini            `toml:"gemini"`
	Openai Openai            `toml:"openai"`
	QWen   QWen              `toml:"qwen"`
	Azure  AzureOpenai       `toml:"azure_openai"`
	Usage  map[string]string `toml:"usage"`
}

func (c *AIConfig) FromENV() {
	c.Usage = make(map[string]string)
	c.Usage["embedding.query"] = os.Getenv("BREW_API_AI_USAGE_E_QUERY")
	c.Usage["embedding.document"] = os.Getenv("BREW_API_AI_USAGE_E_DOCUMENT")
	c.Usage["query"] = os.Getenv("BREW_API_AI_USAGE_QUERY")
	c.Usage["summarize"] = os.Getenv("BREW_API_AI_USAGE_SUMMARIZE")
	c.Usage["enhance_query"] = os.Getenv("BREW_API_AI_USAGE_ENHANCE_QUERY")

	c.Gemini.FromENV()
	c.Openai.FromENV()
	c.Azure.FromENV()
	c.QWen.FromENV()
}

func (c *Gemini) FromENV() {
	c.Token = os.Getenv("BREW_API_AI_GEMINI_TOKEN")
}

func (c *Openai) FromENV() {
	c.Token = os.Getenv("BREW_API_AI_OPENAI_TOKEN")
	c.Endpoint = os.Getenv("BREW_API_AI_OPENAI_ENDPOINT")
}

func (c *AzureOpenai) FromENV() {
	c.Token = os.Getenv("BREW_API_AI_AZURE_OPENAI_TOKEN")
	c.Endpoint = os.Getenv("BREW_API_AI_AZURE_OPENAI_ENDPOINT")
}

func (c *QWen) FromENV() {
	c.Token = os.Getenv("BREW_API_AI_ALI_TOKEN")
	c.Endpoint = os.Getenv("BREW_API_AI_ALI_ENDPOINT")
}

type Gemini struct {
	Token string `toml:"token"`
}

type Openai struct {
	Token          string `toml:"token"`
	Endpoint       string `toml:"endpoint"`
	EmbeddingModel string `toml:"embedding_model"`
	ChatModel      string `toml:"chat_model"`
}

type AzureOpenai struct {
	Token          string `toml:"token"`
	Endpoint       string `toml:"endpoint"`
	EmbeddingModel string `toml:"embedding_model"`
	ChatModel      string `toml:"chat_model"`
}

type QWen struct {
	Token          string `toml:"token"`
	Endpoint       string `toml:"endpoint"`
	EmbeddingModel string `toml:"embedding_model"`
	ChatModel      string `toml:"chat_model"`
}

type AI struct {
	chatDrivers    map[string]ChatAI
	embedDrivers   map[string]EmbeddingAI
	enhanceDrivers map[string]EnhanceAI

	chatUsage    map[string]ChatAI
	enhanceUsage map[string]EnhanceAI
	embedUsage   map[string]EmbeddingAI

	chatDefault    ChatAI
	enhanceDefault EnhanceAI
	embedDefault   EmbeddingAI
}

func (s *AI) NewQuery(ctx context.Context, query []*types.MessageContext) *ai.QueryOptions {
	if d := s.chatUsage["query"]; d != nil {
		return d.NewQuery(ctx, query)
	}
	return s.chatDefault.NewQuery(ctx, query)
}

func (s *AI) Lang() string {
	if d := s.chatUsage["query"]; d != nil {
		return d.Lang()
	}
	return s.chatDefault.Lang()
}

func (s *AI) EmbeddingForQuery(ctx context.Context, content []string) ([][]float32, error) {
	if d := s.embedUsage["embedding.query"]; d != nil {
		return d.EmbeddingForQuery(ctx, content)
	}
	return s.embedDefault.EmbeddingForQuery(ctx, content)
}

func (s *AI) EmbeddingForDocument(ctx context.Context, title string, content []string) ([][]float32, error) {
	if d := s.embedUsage["embedding.document"]; d != nil {
		return d.EmbeddingForDocument(ctx, title, content)
	}
	return s.embedDefault.EmbeddingForQuery(ctx, content)
}

func (s *AI) Summarize(ctx context.Context, doc *string) (ai.SummarizeResult, error) {
	if d := s.chatUsage["summarize"]; d != nil {
		return d.Summarize(ctx, doc)
	}
	return s.chatDefault.Summarize(ctx, doc)
}

func (s *AI) Chunk(ctx context.Context, doc *string) (ai.ChunkResult, error) {
	if d := s.chatUsage["summarize"]; d != nil {
		return d.Chunk(ctx, doc)
	}
	return s.chatDefault.Chunk(ctx, doc)
}

func (s *AI) NewEnhance(ctx context.Context) *ai.EnhanceOptions {
	if d := s.enhanceUsage["enhance_query"]; d != nil {
		return d.NewEnhance(ctx)
	}
	return s.enhanceDefault.NewEnhance(ctx)
}

func (s *AI) MsgIsOverLimit(msgs []*types.MessageContext) bool {
	// TODO
	return false
}

func installAI(a *AI, name string, driver any) {
	if d, ok := driver.(ChatAI); ok {
		a.chatDrivers[name] = d
	}

	if d, ok := driver.(EmbeddingAI); ok {
		a.embedDrivers[name] = d
	}

	if d, ok := driver.(EnhanceAI); ok {
		a.enhanceDrivers[name] = d
	}
}

func SetupAI(cfg AIConfig) (*AI, error) {
	a := &AI{
		chatDrivers:    make(map[string]ChatAI),
		chatUsage:      make(map[string]ChatAI),
		enhanceDrivers: make(map[string]EnhanceAI),
		enhanceUsage:   make(map[string]EnhanceAI),
		embedDrivers:   make(map[string]EmbeddingAI),
		embedUsage:     make(map[string]EmbeddingAI),
	}
	// if cfg.Gemini.Token != "" {
	// 	a.drivers[gemini.NAME] = gemini.New(cfg.Lang, cfg.Gemini.Token)
	// 	a._default = a.drivers[gemini.NAME]
	// }
	if cfg.Openai.Token != "" {
		var oai any
		oai = openai.New(cfg.Openai.Token, cfg.Openai.Endpoint, ai.ModelName{
			ChatModel:      cfg.Openai.ChatModel,
			EmbeddingModel: cfg.Openai.EmbeddingModel,
		})

		installAI(a, openai.NAME, oai)
	}

	if cfg.Azure.Token != "" {
		var oai any
		oai = azure_openai.New(cfg.Azure.Token, cfg.Azure.Endpoint, ai.ModelName{
			ChatModel:      cfg.Azure.ChatModel,
			EmbeddingModel: cfg.Azure.EmbeddingModel,
		})

		installAI(a, azure_openai.NAME, oai)
	}

	if cfg.QWen.Token != "" {
		var oai any
		oai = qwen.New(cfg.QWen.Token, cfg.QWen.Endpoint, ai.ModelName{
			ChatModel:      cfg.QWen.ChatModel,
			EmbeddingModel: cfg.QWen.EmbeddingModel,
		})

		installAI(a, qwen.NAME, oai)
	}

	for k, v := range cfg.Usage {
		if strings.Contains(k, "embedding") {
			a.embedUsage[k] = a.embedDrivers[v]
		} else {
			a.chatUsage[k] = a.chatDrivers[v]
		}
	}

	for _, v := range a.chatDrivers {
		a.chatDefault = v
		break
	}

	for _, v := range a.embedDrivers {
		a.embedDefault = v
		break
	}

	for _, v := range a.enhanceDrivers {
		a.enhanceDefault = v
		break
	}

	if a.chatDefault == nil || a.embedDefault == nil {
		panic("AI driver of chat and embedding must be set")
	}

	return a, nil
}

type ApplyFunc func(s *Srv)

func ApplyAI(cfg AIConfig) ApplyFunc {
	return func(s *Srv) {
		s.ai, _ = SetupAI(cfg)
	}
}
