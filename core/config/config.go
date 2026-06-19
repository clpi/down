package config

type Settings = map[string]interface{}

type (
	DownSettings     struct{}
	MarkdownSettings struct{}
)

var DocdownSettings = Settings{}

var (
	DownConfig = Settings{
		"down": Settings{
			"markdown": Settings{
				"enabled": true,
				"completion": Settings{
					"enabled": true,
				},
			},
		},
	}
	DocdownConfig = Settings{
		"enabled": true,
		"codeAction": Settings{
			"enabled": true,
		},
		"codeLens": Settings{
			"enabled": true,
		},
		"inlayHint": Settings{
			"enabled": true,
		},
		"completion": Settings{
			"enabled": true,
		},
	}
	MarkdownConfig = Settings{
		"enabled": true,
		"codeAction": Settings{
			"enabled": true,
		},
		"codeLens": Settings{
			"enabled": true,
		},
		"inlayHint": Settings{
			"enabled": true,
		},
		"completion": Settings{
			"enabled": true,
		},
	}
)
