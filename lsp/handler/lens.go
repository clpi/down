package handler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/clpi/down/lsp/files"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var reLensTask = regexp.MustCompile(`^\s*[-*+]\s+\[([ xX])\]\s+(.*)`)

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

func (s *State) CodeLens(_ *glsp.Context, p *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return s.workspaceLenses(), nil
	}

	var lens []protocol.CodeLens
	lens = append(lens, s.workspaceLenses()...)

	openCount := 0
	doneCount := 0
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		m := reLensTask.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		completed := m[1] != " "
		if completed {
			doneCount++
		} else {
			openCount++
		}

		title := "Toggle task"
		if completed {
			title = "Mark incomplete"
		}
		lineNum := i
		lens = append(lens, protocol.CodeLens{
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
				End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
			},
			Command: &protocol.Command{
				Command:   "down.task.toggle",
				Title:     title,
				Arguments: []any{uri, lineNum},
			},
		})
	}

	if openCount+doneCount > 0 {
		summary := fmt.Sprintf("Tasks: %d open, %d done", openCount, doneCount)
		lens = append(lens, protocol.CodeLens{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Command: &protocol.Command{
				Command:   "down.task.list",
				Title:     summary,
				Arguments: []any{uri},
			},
		})
	}

	if s.Graph != nil {
		entities := s.Graph.EntitiesByDocument(uri)
		if len(entities) > 0 {
			title := fmt.Sprintf("Knowledge: %d entities tracked", len(entities))
			lens = append(lens, protocol.CodeLens{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Command: &protocol.Command{
					Command: "down.knowledge.summary",
					Title:   title,
				},
			})
		}

		tagCount := 0
		for _, ent := range entities {
			if ent.Kind == knowledge.KindTag {
				tagCount++
			}
		}
		if tagCount > 0 {
			title := fmt.Sprintf("Tags: %d in document", tagCount)
			lens = append(lens, protocol.CodeLens{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Command: &protocol.Command{
					Command:   "down.knowledge.entities",
					Title:     title,
					Arguments: []any{"tag"},
				},
			})
		}
	}

	return lens, nil
}

func (s *State) workspaceLenses() []protocol.CodeLens {
	return []protocol.CodeLens{
		{
			Command: &protocol.Command{
				Command: "down.workspace.open",
				Title:   "Open workspace",
			},
		},
		{
			Command: &protocol.Command{
				Command: "down.workspace.list",
				Title:   "List workspaces",
			},
		},
		{
			Command: &protocol.Command{
				Command: "down.workspace.new",
				Title:   "New workspace",
			},
		},
	}
}

func (s *State) LensResolve(_ *glsp.Context, p *protocol.CodeLens) (*protocol.CodeLens, error) {
	return p, nil
}
