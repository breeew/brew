package jina_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/starbx/brew-api/pkg/ai/jina"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

func new() *jina.Driver {
	return jina.New(os.Getenv("BREW_API_AI_JINA_TOKEN"))
}

func Test_Reader(t *testing.T) {
	d := new()

	resp, err := d.Reader(context.Background(), "https://brew.re")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*resp)
}
