package ai

import (
	"context"
	"net/http"
	"os"
)

type CloudflareProvider struct {
	apiToken  string
	accountID string
	model     string
	client    *http.Client
}

func NewCloudflareProvider() *CloudflareProvider {
	model := os.Getenv("DOWN_CF_MODEL")
	if model == "" {
		model = "@cf/meta/llama-3.1-8b-instruct"
	}
	return &CloudflareProvider{
		apiToken:  os.Getenv("CLOUDFLARE_API_TOKEN"),
		accountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		model:     model,
		client:    httpClient(),
	}
}

func (p *CloudflareProvider) Name() string   { return "cloudflare" }
func (p *CloudflareProvider) Available() bool { return p.apiToken != "" && p.accountID != "" }

func (p *CloudflareProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	endpoint := "https://api.cloudflare.com/client/v4/accounts/" + p.accountID + "/ai/v1/chat/completions"

	return openAIComplete(ctx, openAIProviderConfig{
		endpoint: endpoint,
		model:    p.model,
		headers:  map[string]string{"Authorization": "Bearer " + p.apiToken},
		client:   p.client,
	}, req)
}
