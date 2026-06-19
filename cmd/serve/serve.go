package serve

import (
	"log"

	cmdutil "github.com/clpi/down/cmd/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// func (s *Server) serve(a []string) {
// 	log.Println("ZServe")
// }

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
		log.Println("Serve")
	}
)

var Serve = cmdutil.Cmd("serve", serveS, "serve", serveS[0], serveR)
