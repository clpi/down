package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GeminiProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewGeminiProvider() *GeminiProvider {
	model := os.Getenv("DOWN_GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	baseURL := os.Getenv("DOWN_GEMINI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return &GeminiProvider{
		apiKey:  os.Getenv("GEMINI_API_KEY"),
		model:   model,
		baseURL: baseURL,
		client:  httpClient(),
	}
}

func (p *GeminiProvider) Name() string   { return "gemini" }
func (p *GeminiProvider) Available() bool { return p.apiKey != "" }

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenConfig `json:"generationConfig,omitempty"`
}

type geminiGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.Available() {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	contents := make([]geminiContent, len(req.Messages))
	for i, m := range req.Messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents[i] = geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		}
	}

	body := geminiRequest{
		Contents: contents,
		GenerationConfig: &geminiGenConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     req.Temperature,
		},
	}

	if req.SystemPrompt != "" {
		body.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)

	respBody, err := doJSON(ctx, p.client, "POST", url, nil, body)
	if err != nil {
		return nil, err
	}

	var result geminiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return &CompletionResponse{
		Text: result.Candidates[0].Content.Parts[0].Text,
		Usage: Usage{
			InputTokens:  result.UsageMetadata.PromptTokenCount,
			OutputTokens: result.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}
