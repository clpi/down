package config

import (
	"log"

	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

type (
	WorkspaceConfig struct {
		Name      string
		Path      string
		Index     string
		Log       LogConfig
		Snippets  SnippetsConfig
		Templates TemplatesConfig
		Notes     NotesConfig
		Data      DataConfig
	}
	DataConfig struct {
		Dir string
	}
	LogConfig struct {
		Dir string
	}
	SnippetsConfig struct {
		Dir string
	}
	TemplatesConfig struct {
		Dir string
	}
	NotesConfig struct {
		Dir string
	}
)

var (
	Config = cobra.Command{
		Use: "config <command>",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   false,
			DisableDescriptions: false,
			HiddenDefaultCmd:    false,
			DisableNoDescFlag:   true,
		},
		DisableSuggestions: false,
		Example:            "config",
		Version:            lsp.Version,
		Aliases:            []string{"cfg", "conf"},
		Long:               "config",
		Hidden:                true,
		Short:              "c",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("config")
		},
	}
)
