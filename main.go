package main

import (
	"context"
	cmd "github.com/clpi/down/cmd"
	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
)

func main() {
	commonlog.Configure(2, nil)
	ctx := context.Background()
	cmd.Run(&ctx)
}
