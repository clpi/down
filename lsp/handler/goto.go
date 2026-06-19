package handler

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TypeDefinition implements textDocument/typeDefinition.
// For markdown, the "type" of a tag or wiki-link target is its canonical note/heading.
// This resolves a tag reference to the document where it is first defined or most mentioned.
func (s *State) TypeDefinition(_ *glsp.Context, p *protocol.TypeDefinitionParams) (any, error) {
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

	// Try tags first (#tag), then wiki links ([[target]]), then entities.
	target := word
	if col > 0 && line[col-1] == '#' {
		target = word
	}

	results := s.Graph.Search(target)
	var locations []protocol.Location
	for _, ent := range results {
		if !strings.EqualFold(ent.Name, target) {
			continue
		}
		// Pick the source with the most mentions, or the earliest if tied.
		var best *protocol.Location
		bestScore := -1
		for _, src := range ent.Sources {
			score := ent.Mentions
			if best == nil || score > bestScore {
				bestScore = score
				best = &protocol.Location{
					URI: protocol.DocumentUri(src.URI),
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(src.Line), Character: 0},
						End:   protocol.Position{Line: protocol.UInteger(src.Line), Character: protocol.UInteger(len(ent.Name))},
					},
				}
			}
		}
		if best != nil {
			locations = append(locations, *best)
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

// Implementation implements textDocument/implementation.
// For markdown documents, an "implementation" is a document that materially
// implements/contains the concept referenced at the cursor.
func (s *State) Implementation(_ *glsp.Context, p *protocol.ImplementationParams) (any, error) {
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
	seen := make(map[string]bool)
	var locations []protocol.Location
	for _, ent := range results {
		if !strings.EqualFold(ent.Name, word) {
			continue
		}
		for _, src := range ent.Sources {
			if src.URI == uri {
				continue
			}
			key := src.URI + ":" + intStr(src.Line)
			if seen[key] {
				continue
			}
			seen[key] = true
			locations = append(locations, protocol.Location{
				URI: protocol.DocumentUri(src.URI),
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(src.Line), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(src.Line), Character: protocol.UInteger(len(ent.Name))},
				},
			})
		}
	}

	if len(locations) == 0 {
		return nil, nil
	}
	if len(locations) == 1 {
		return locations[0], nil
	}
	return locations, nil
}
