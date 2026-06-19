package serve

import (
	"log"

	ls "github.com/clpi/down/lsp"
	cmdutil "github.com/clpi/down/cmd/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	serveS  = []string{"srv", "start"}
	serveFs = pflag.FlagSet{}
	serveO  = []pflag.Flag{}
	serveF  = []pflag.Flag{
		{
			Name:      "host",
			Hidden:    false,
			Value:     nil,
			Usage:     "host",
			Shorthand: "h",
			DefValue:  "0.0.0.0",
		},
		{
			Name:      "port",
			Hidden:    false,
			Value:     nil,
			Usage:     "port",
			Shorthand: "p",
			DefValue:  "8844",
		},
	}
	serveR = func(cmd *cobra.Command, args []string) {
		lsp, err := ls.NewServer()
		if err != nil {
			log.Fatal(err)
		}
		lsp.Server.RunStdio()
	}
)

var Serve = cmdutil.Cmd("serve", serveS, "serve", serveS[0], serveR)
