package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// openAIChatRequest is the standard OpenAI-compatible chat completion format
// used by Ollama, xAI, Cloudflare Workers AI, and others.
type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func openAIComplete(ctx context.Context, cfg openAIProviderConfig, req CompletionRequest) (*CompletionResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	msgs := make([]Message, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		msgs = append(msgs, Message{Role: "system", Content: req.SystemPrompt})
	}
	msgs = append(msgs, req.Messages...)

	temp := req.Temperature
	if temp == 0 {
		temp = 0.7
	}

	body := openAIChatRequest{
		Model:       cfg.model,
		Messages:    msgs,
		MaxTokens:   maxTokens,
		Temperature: temp,
		Stream:      false,
	}

	respBody, err := doJSON(ctx, cfg.client, "POST", cfg.endpoint, cfg.headers, body)
	if err != nil {
		return nil, err
	}

	var result openAIChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return &CompletionResponse{
		Text: result.Choices[0].Message.Content,
		Usage: Usage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
		},
	}, nil
}
