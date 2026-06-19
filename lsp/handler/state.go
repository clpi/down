package handler

import (
	"github.com/clpi/down/core/workspace"
	"github.com/clpi/down/lsp/ai"
	"github.com/clpi/down/lsp/knowledge"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

type (
	Session map[string]interface{}
	State   struct {
		Session     Session
		Server      *server.Server
		Workspaces  map[string]workspace.Workspace
		Diagnostics []protocol.Diagnostic
		Graph       *knowledge.Graph
		AI          *ai.Engine
		Documents   map[string]string
	}
	LoadData struct {
		Config  interface{} `json:"settings,omitempty"`
		Folders []protocol.WorkspaceFolder
		Symbols []protocol.SymbolInformation
		Tags    []protocol.DocumentSymbol
		Links   []protocol.DocumentLink
	}
)
