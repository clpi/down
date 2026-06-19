package handler

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Declaration implements textDocument/declaration.
// For markdown documents, a "declaration" is the first occurrence of a knowledge
// graph entity (tag, person, concept, project, etc.) across the workspace.
func (s *State) Declaration(_ *glsp.Context, p *protocol.DeclarationParams) (any, error) {
	if s.Graph == nil {
		return nil, nil
	}

	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil, nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col >= len(line) {
		return nil, nil
	}

	word := wordAtPosition(line, col)
	if word == "" {
		return nil, nil
	}

	results := s.Graph.Search(word)
	var locations []protocol.Location
	for _, ent := range results {
		if !strings.EqualFold(ent.Name, word) {
			continue
		}
		for _, src := range ent.Sources {
			// Declaration is the earliest mention across all documents.
			locations = append(locations, protocol.Location{
				URI: protocol.DocumentUri(src.URI),
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(src.Line), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(src.Line), Character: protocol.UInteger(len(ent.Name))},
				},
			})
			break
		}
		break
	}

	if len(locations) == 0 {
		return nil, nil
	}
	if len(locations) == 1 {
		return locations[0], nil
	}
	return locations, nil
}
