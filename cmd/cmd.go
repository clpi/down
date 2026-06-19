package cmd

import (
    "context"
    "log"

    "github.com/clpi/down/cmd/add"
    "github.com/clpi/down/cmd/compact"
    "github.com/clpi/down/cmd/config"
    downctx "github.com/clpi/down/cmd/context"
    "github.com/clpi/down/cmd/delete"
    "github.com/clpi/down/cmd/export"
    "github.com/clpi/down/cmd/find"
    "github.com/clpi/down/cmd/initialize"
    "github.com/clpi/down/cmd/list"
    logc "github.com/clpi/down/cmd/log"
    lsc "github.com/clpi/down/cmd/lsp"
    "github.com/clpi/down/cmd/mcp"
    "github.com/clpi/down/cmd/memory"
    "github.com/clpi/down/cmd/new"
    "github.com/clpi/down/cmd/note"
    "github.com/clpi/down/cmd/profile"
    "github.com/clpi/down/cmd/publish"
    "github.com/clpi/down/cmd/remove"
    "github.com/clpi/down/cmd/serve"
    "github.com/clpi/down/cmd/shell"
    "github.com/clpi/down/cmd/skills"
    "github.com/clpi/down/cmd/sync"
    cmdutil "github.com/clpi/down/cmd/util"
    "github.com/clpi/down/cmd/workspace"
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
)

func flag() (cmd *cobra.Command, f pflag.Flag) {
	return
}

var downR = func(cmd *cobra.Command, args []string) {
	log.Println(`down`)
}

var Down = cmdutil.Cmd("down", []string{"d"}, "down", "down", downR)

func Configure() {
    cobra.EnableCommandSorting = true
    cobra.EnablePrefixMatching = true
    Down.AddCommand(&lsc.Lsp)
    Down.AddCommand(&initialize.Init)
    Down.AddCommand(&Runc)
    Down.AddCommand(&workspace.Workspace)
    Down.AddCommand(&find.Find)
    Down.AddCommand(&list.List)
    Down.AddCommand(&config.Config)
    Down.AddCommand(&logc.Log)
    Down.AddCommand(&Tag)
    Down.AddCommand(&new.New)
    Down.AddCommand(&note.Note)
    Down.AddCommand(&Link)
    Down.AddCommand(&shell.Shell)
    Down.AddCommand(&serve.Serve)
    Down.AddCommand(&delete.Delete)
    Down.AddCommand(&export.Export)
    Down.AddCommand(&sync.Sync)
    Down.AddCommand(&Snippet)
    Down.AddCommand(&Template)
    Down.AddCommand(&profile.Profile)
    Down.AddCommand(&mcp.Mcp)
    Down.AddCommand(&compact.Compact)
    Down.AddCommand(&skills.Skills)
    Down.AddCommand(&add.Add)
    Down.AddCommand(&remove.Remove)
    Down.AddCommand(&memory.Memory)
    Down.AddCommand(&downctx.Context)
    Down.AddCommand(&publish.Publish)
}

func Run(c *context.Context) {
	Configure()
	Down.Execute()
}
