package workspace

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var DownSettings map[string]interface{} = map[string]interface{}{
	"down": map[string]interface{}{
		"codeAction": map[string]interface{}{
			"enabled": true,
		},
		"codeLens": map[string]interface{}{
			"enabled": true,
		},
		"inlayHint": map[string]interface{}{
			"enabled": true,
		},
		"completion": map[string]interface{}{
			"enabled": true,
		},
		"ai": map[string]interface{}{
			"enabled":     true,
			"provider":    "auto",
			"completions": true,
		},
		"knowledge": map[string]interface{}{
			"enabled":   true,
			"autoIndex": true,
		},
		"diagnostics": map[string]interface{}{
			"enabled":         true,
			"brokenLinks":     true,
			"unresolvedLinks": true,
			"overdueTasks":    true,
		},
		"formatting": map[string]interface{}{
			"enabled":             true,
			"trimTrailingSpaces":  true,
			"ensureFinalNewline":  true,
			"collapseBlankLines":  true,
		},
		"enabled": true,
	},
	"markdown": map[string]interface{}{
		"enabled": true,
		"completion": map[string]interface{}{
			"enabled": true,
		},
	},
}

// Configure processes configuration change notifications from the client.
func Configure(c *glsp.Context, p *protocol.DidChangeConfigurationParams) error {
	settings := map[string]interface{}{
		"markdown": map[string]interface{}{},
		"down": map[string]interface{}{
			"enabled": true,
			"codeAction": map[string]interface{}{
				"enabled": true,
			},
			"codeLens": map[string]interface{}{
				"enabled": true,
			},
			"inlayHint": map[string]interface{}{
				"enabled": true,
			},
			"diagnostics": map[string]interface{}{
				"enabled": true,
			},
			"hover": map[string]interface{}{
				"enabled": true,
			},
			"completion": map[string]interface{}{
				"enabled": true,
			},
			"signatureHelp": map[string]interface{}{
				"enabled": true,
			},
			"ai": map[string]interface{}{
				"enabled":     true,
				"provider":    "auto",
				"completions": true,
			},
			"knowledge": map[string]interface{}{
				"enabled":   true,
				"autoIndex": true,
			},
		},
	}
	p.Settings = settings
	return nil
}
