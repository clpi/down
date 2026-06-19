package handler

import (
	"fmt"

	"github.com/clpi/down/lsp/files"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	LensProvider = protocol.CodeLensOptions{
		ResolveProvider: &trueVal,
		WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
			WorkDoneProgress: &trueVal,
		},
	}
	LensRegistration = protocol.CodeLensRegistrationOptions{
		TextDocumentRegistrationOptions: protocol.TextDocumentRegistrationOptions{
			DocumentSelector: &files.Filetypes,
		},
		CodeLensOptions: LensProvider,
	}
)

func (s *State) CodeLens(c *glsp.Context, p *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	var lens []protocol.CodeLens
	var (
		wsOpen protocol.CodeLens = protocol.CodeLens{
			Data: nil,
			Command: &protocol.Command{
				Command:   "down.workspace.open",
				Arguments: nil,
				Title:     "Open workspace",
			},
		}
		wsNew protocol.CodeLens = protocol.CodeLens{
			Data: nil,
			Command: &protocol.Command{
				Arguments: nil,
				Title:     "new workspace",
				Command:   "down.workspace.new",
			},
		}
	)
	lens = append(lens, wsOpen, wsNew)

	if s.Graph != nil {
		uri := string(p.TextDocument.URI)
		entities := s.Graph.EntitiesByDocument(uri)
		if len(entities) > 0 {
			title := fmt.Sprintf("Knowledge: %d entities tracked", len(entities))
			lens = append(lens, protocol.CodeLens{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Command: &protocol.Command{
					Command:   "down.knowledge.summary",
					Title:     title,
				},
			})
		}
	}

	return lens, nil
}

func (s *State) LensResolve(c *glsp.Context, p *protocol.CodeLens) (*protocol.CodeLens, error) {
	return p, nil
}
