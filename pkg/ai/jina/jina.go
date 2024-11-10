package jina

// provider for https://jina.ai/
// - reader

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/breeew/brew-api/pkg/ai"
)

type Driver struct {
	client *http.Client
	token  string
}

const (
	NAME = "jina"
)

func New(token string) *Driver {
	return &Driver{
		client: &http.Client{},
		token:  token,
	}
}

func (s *Driver) appleBaseHeader(req *http.Request) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+s.token)
}

func (s *Driver) Reader(ctx context.Context, endpoint string) (*ai.ReaderResult, error) {
	slog.Debug("Reader", slog.String("driver", NAME))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://r.jina.ai/"+endpoint, nil)
	s.appleBaseHeader(req)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to request jina reader: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &ai.ReaderResult{
		Content: string(body),
	}, nil
}
