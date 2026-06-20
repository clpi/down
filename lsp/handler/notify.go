package handler

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// postDocumentChange republishes diagnostics and asks the client to refresh semantic tokens.
func (s *State) postDocumentChange(ctx *glsp.Context, uri, text string) {
	if ctx == nil {
		return
	}
	s.publishDiagnostics(ctx, uri, text)
	s.requestSemanticRefresh(ctx)
}

func (s *State) requestSemanticRefresh(ctx *glsp.Context) {
	if ctx == nil || ctx.Call == nil {
		return
	}
	var resp any
	ctx.Call(string(protocol.MethodWorkspaceSemanticTokensRefresh), nil, &resp)
}

func (s *State) applyWorkspaceEdit(ctx *glsp.Context, label string, edit protocol.WorkspaceEdit) protocol.ApplyWorkspaceEditResponse {
	var resp protocol.ApplyWorkspaceEditResponse
	if ctx == nil || ctx.Call == nil {
		return resp
	}
	params := protocol.ApplyWorkspaceEditParams{
		Label: &label,
		Edit:  edit,
	}
	ctx.Call(string(protocol.ServerWorkspaceApplyEdit), params, &resp)
	return resp
}
