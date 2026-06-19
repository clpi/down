package state

import (
	"github.com/clpi/down/core/store"
	"github.com/clpi/down/core/workspace"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type (
	Documents struct {
		documents map[protocol.DocumentUri]*protocol.TextDocumentItem
	}
	State struct {
		Workspaces  workspace.Workspaces
		Diagnostics map[protocol.DocumentUri][]protocol.Diagnostic
		Metadata    store.Store[string, interface{}]
		Documents   *Documents
		Logger      commonlog.Logger
	}
)
