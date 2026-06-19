package ai

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type openAIProviderConfig struct {
	endpoint string
	model    string
	headers  map[string]string
	client   *http.Client
}

type ProviderInfo struct {
	Name      string
	Available bool
	EnvVars   []string
}

func AllProviders() []Provider {
	return []Provider{
		NewAnthropicProvider(),
		NewOllamaProvider(),
		NewGeminiProvider(),
		NewXAIProvider(),
		NewCloudflareProvider(),
	}
}

func ProviderStatus() []ProviderInfo {
	providers := AllProviders()
	info := make([]ProviderInfo, len(providers))
	envMap := map[string][]string{
		"anthropic":  {"ANTHROPIC_API_KEY", "DOWN_ANTHROPIC_MODEL"},
		"ollama":     {"DOWN_OLLAMA_BASE_URL", "DOWN_OLLAMA_MODEL"},
		"gemini":     {"GEMINI_API_KEY", "DOWN_GEMINI_MODEL"},
		"xai":        {"XAI_API_KEY", "DOWN_XAI_MODEL"},
		"cloudflare": {"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID", "DOWN_CF_MODEL"},
	}
	for i, p := range providers {
		info[i] = ProviderInfo{
			Name:      p.Name(),
			Available: p.Available(),
			EnvVars:   envMap[p.Name()],
		}
	}
	return info
}

// SelectProvider picks a provider based on DOWN_AI_PROVIDER env var,
// or auto-selects the first available one.
// Priority order: explicit env > anthropic > ollama > gemini > xai > cloudflare
func SelectProvider() (Provider, error) {
	explicit := os.Getenv("DOWN_AI_PROVIDER")
	if explicit != "" {
		explicit = strings.ToLower(explicit)
		for _, p := range AllProviders() {
			if p.Name() == explicit {
				if !p.Available() {
					return nil, fmt.Errorf("provider %q selected but not available (check env vars)", explicit)
				}
				return p, nil
			}
		}
		return nil, fmt.Errorf("unknown provider %q; available: anthropic, ollama, gemini, xai, cloudflare", explicit)
	}

	for _, p := range AllProviders() {
		if p.Available() {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no AI provider available; set one of: ANTHROPIC_API_KEY, GEMINI_API_KEY, XAI_API_KEY, CLOUDFLARE_API_TOKEN, or run Ollama locally")
}

func ProviderSummary() string {
	var sb strings.Builder
	sb.WriteString("AI Providers:\n")
	for _, info := range ProviderStatus() {
		status := "unavailable"
		if info.Available {
			status = "available"
		}
		sb.WriteString(fmt.Sprintf("  %s: %s (env: %s)\n", info.Name, status, strings.Join(info.EnvVars, ", ")))
	}
	return sb.String()
}
