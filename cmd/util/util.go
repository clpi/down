package util

import (
	"github.com/spf13/cobra"
	// "github.com/tliron/commonlog"
)

var (
	cmds = map[string]cobra.Command{
		"serve": {
			Run: func(cmd *cobra.Command, args []string) {},
			Use: `serve`,
		},
		"start": {
			Run: func(cmd *cobra.Command, args []string) {},
			Use: `start`,
		},
	}
	env = map[string]string{
		"DOWN_BIN_DIR":    "~/.local/bin",
		"DOWN_CONFIG_DIR": "~/.config/down",
		"DOOM_WORKSPACES": "~/down:~/notes:~/work",
		"DOWN_CACHE_DIR":  "~/.cache/down",
		"DOWN_LOG_DIR":    "~/.cache/down",
	}
	envl = map[string][]string{
		"DOOM_WORKSPACE_PATHS": {
			"~/down",
			"~/notes",
			"~/work",
		},
	}
	envm = map[string]map[string]string{
		"DOWN_WORKSPACES_CONFIG": {
			"default": "~/down",
			"notes":   "~/notes",
			"work":    "~/work",
		},
	}
	envc = map[string]map[string]map[string]interface{}{
		"DOWN_WORKSPACES_CONFIG": {
			"default": {
				"name":  "default",
				"path":  "~/down",
				"index": "index",
				"templates": map[string]string{
					"dir": "templates",
				},
				"snippets": map[string]string{
					"dir": "snippets",
				},
				"notes": map[string]string{
					"dir": "notes",
				},
				"log": map[string]string{
					"dir": "log",
				},
				"config": map[string]string{
					"index_file": "index",
				},
			},
		},
	}
)
