package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Name() string
	Available() bool
}

type CompletionRequest struct {
	SystemPrompt string
	Messages     []Message
	MaxTokens    int
	Temperature  float64
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionResponse struct {
	Text  string
	Usage Usage
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

func doJSON(ctx context.Context, client *http.Client, method, url string, headers map[string]string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// AnthropicProvider uses the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewAnthropicProvider() *AnthropicProvider {
	model := os.Getenv("DOWN_ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	baseURL := os.Getenv("DOWN_ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &AnthropicProvider{
		apiKey:  os.Getenv("ANTHROPIC_API_KEY"),
		model:   model,
		baseURL: baseURL,
		client:  httpClient(),
	}
}

func (p *AnthropicProvider) Name() string      { return "anthropic" }
func (p *AnthropicProvider) Available() bool    { return p.apiKey != "" }

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.Available() {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	msgs := make([]msg, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = msg{Role: m.Role, Content: m.Content}
	}

	body := struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		System    string `json:"system,omitempty"`
		Messages  []msg  `json:"messages"`
	}{
		Model:     p.model,
		MaxTokens: maxTokens,
		System:    req.SystemPrompt,
		Messages:  msgs,
	}

	respBody, err := doJSON(ctx, p.client, "POST", p.baseURL+"/v1/messages",
		map[string]string{
			"x-api-key":          p.apiKey,
			"anthropic-version":  "2023-06-01",
		}, body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return &CompletionResponse{
		Text: result.Content[0].Text,
		Usage: Usage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
		},
	}, nil
}
