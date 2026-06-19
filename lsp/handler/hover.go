package handler

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *State) Hover(c *glsp.Context, p *protocol.HoverParams) (*protocol.Hover, error) {
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

	wordStart, wordEnd := col, col
	for wordStart > 0 && isWordChar(line[wordStart-1]) {
		wordStart--
	}
	for wordEnd < len(line) && isWordChar(line[wordEnd]) {
		wordEnd++
	}
	if wordStart >= wordEnd {
		return nil, nil
	}
	word := line[wordStart:wordEnd]

	if s.Graph == nil || len(word) < 2 {
		return nil, nil
	}

	results := s.Graph.Search(word)
	if len(results) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	for _, ent := range results {
		sb.WriteString("### " + ent.Name + "\n")
		sb.WriteString("**Type**: " + string(ent.Kind) + "  \n")
		sb.WriteString("**Mentions**: " + intStr(ent.Mentions) + "  \n")

		if len(ent.Properties) > 0 {
			sb.WriteString("\n**Properties**:\n")
			for k, v := range ent.Properties {
				sb.WriteString("- " + k + ": " + v + "\n")
			}
		}

		rels := s.Graph.RelationsFrom(ent.ID)
		if len(rels) > 0 {
			sb.WriteString("\n**Relations**:\n")
			for _, r := range rels {
				if target, exists := s.Graph.Entities[r.To]; exists {
					sb.WriteString("- " + string(r.Kind) + " → " + target.Name + "\n")
				}
			}
		}

		backRels := s.Graph.RelationsTo(ent.ID)
		if len(backRels) > 0 {
			sb.WriteString("\n**Referenced by**:\n")
			for _, r := range backRels {
				if source, exists := s.Graph.Entities[r.From]; exists {
					sb.WriteString("- " + source.Name + " " + string(r.Kind) + "\n")
				}
			}
		}
		sb.WriteString("\n---\n")
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: sb.String() + s.BacklinksSummary(uri),
		},
		Range: &protocol.Range{
			Start: protocol.Position{Line: p.Position.Line, Character: protocol.UInteger(wordStart)},
			End:   protocol.Position{Line: p.Position.Line, Character: protocol.UInteger(wordEnd)},
		},
	}, nil
}

