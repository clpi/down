package handler

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *State) DocumentHighlight(_ *glsp.Context, p *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
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
	word := strings.ToLower(line[wordStart:wordEnd])
	if len(word) < 2 {
		return nil, nil
	}

	var highlights []protocol.DocumentHighlight
	kindText := protocol.DocumentHighlightKindText
	kindWrite := protocol.DocumentHighlightKindWrite

	for i, ln := range lines {
		lnLower := strings.ToLower(ln)
		searchFrom := 0
		for {
			idx := strings.Index(lnLower[searchFrom:], word)
			if idx < 0 {
				break
			}
			absIdx := searchFrom + idx
			before := absIdx == 0 || !isWordChar(ln[absIdx-1])
			after := absIdx+len(word) >= len(ln) || !isWordChar(ln[absIdx+len(word)])
			if before && after {
				kind := &kindText
				if strings.HasPrefix(strings.TrimSpace(ln), "#") {
					kind = &kindWrite
				}
				highlights = append(highlights, protocol.DocumentHighlight{
					Kind: kind,
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(absIdx)},
						End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(absIdx + len(word))},
					},
				})
			}
			searchFrom = absIdx + len(word)
		}
	}
	return highlights, nil
}
