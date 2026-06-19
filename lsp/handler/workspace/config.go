package handler

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
		"enabled": true,
	},
	"markdown": map[string]interface{}{
		"enabled": true,
		"completion": map[string]interface{}{
			"enabled": true,
		},
	},
	"docdown": map[string]interface{}{
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
		"completion": map[string]interface{}{
			"enabled": true,
		},
	},
}
