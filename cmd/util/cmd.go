package util

import (
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	// "github.com/tliron/commonlog"
)

var (
	FParseErrWhitelist = cobra.FParseErrWhitelist{
		UnknownFlags: true,
	}
	CompleteOpts = cobra.CompletionOptions{
		DisableDefaultCmd:   false,
		DisableDescriptions: false,
		HiddenDefaultCmd:    false,
		DisableNoDescFlag:   true,
	}
)

func Cmd(l string, sh []string, u string, e string, r func(*cobra.Command, []string)) cobra.Command {
	return cobra.Command{
		Example: l,
		Version: lsp.Version,
		CompletionOptions:     CompleteOpts,
		DisableFlagsInUseLine: false,
		Hidden:                false,
		DisableSuggestions:    false,
		DisableAutoGenTag:     false,
		DisableFlagParsing:    false,
		SilenceUsage:          false,
		FParseErrWhitelist:    FParseErrWhitelist,
		Annotations: map[string]string{
			"a": "b",
		},
		SilenceErrors: false,
		Aliases:       sh,
		Use:           l,
		Long:          l,
		Short:         sh[0],
		Run:           r,
	}
}

func AddFlag(c *cobra.Command, f pflag.Flag) {
}
