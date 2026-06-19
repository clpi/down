package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider() *OllamaProvider {
	baseURL := os.Getenv("DOWN_OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := os.Getenv("DOWN_OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  httpClient(),
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) Available() bool {
	resp, err := http.Get(p.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func (p *OllamaProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return openAIComplete(ctx, openAIProviderConfig{
		endpoint: p.baseURL + "/v1/chat/completions",
		model:    p.model,
		headers:  nil,
		client:   p.client,
	}, req)
}

// ListModels returns the models available on the Ollama instance.
func (p *OllamaProvider) ListModels() ([]string, error) {
	resp, err := http.Get(p.baseURL + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse model list: %w", err)
	}

	names := make([]string, len(result.Models))
	for i, m := range result.Models {
		names[i] = m.Name
	}
	return names, nil
}
