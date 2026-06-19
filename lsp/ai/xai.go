package ai

import (
	"context"
	"net/http"
	"os"
)

type XAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewXAIProvider() *XAIProvider {
	model := os.Getenv("DOWN_XAI_MODEL")
	if model == "" {
		model = "grok-3-mini-fast"
	}
	baseURL := os.Getenv("DOWN_XAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.x.ai"
	}
	return &XAIProvider{
		apiKey:  os.Getenv("XAI_API_KEY"),
		model:   model,
		baseURL: baseURL,
		client:  httpClient(),
	}
}

func (p *XAIProvider) Name() string   { return "xai" }
func (p *XAIProvider) Available() bool { return p.apiKey != "" }

func (p *XAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return openAIComplete(ctx, openAIProviderConfig{
		endpoint: p.baseURL + "/v1/chat/completions",
		model:    p.model,
		headers:  map[string]string{"Authorization": "Bearer " + p.apiKey},
		client:   p.client,
	}, req)
}
