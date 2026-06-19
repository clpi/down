package store

import "go.lsp.dev/protocol"

type (
	InitConfig struct {
		Git struct {
			Enabled bool `json:"enabled"`
		}
		Sync struct {
			Enabled bool `json:"enabled"`
		}
		Markdown struct {
			Enabled bool `json:"enabled"`
		}
		Command []struct {
			Enabled bool     `json:"enabled"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
	}
	GlobalConfig struct {
		Uri      protocol.URI           `json:"uri"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	LocalConfig[Config any] struct {
		Uri  protocol.URI `json:"uri"`
		Data Config       `json:"data"`
	}
)
