package handler

import (
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var reLinkedWiki = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// LinkedEditing returns ranges that should be edited together.
// For wiki links [[target]], editing the target text inside one occurrence
// should update all other occurrences of the same link in the document.
func (s *State) LinkedEditing(_ *glsp.Context, p *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	line := int(p.Position.Line)
	col := int(p.Position.Character)
	if line >= len(lines) {
		return nil, nil
	}

	// Check if cursor is inside a [[wiki link]]
	var targetName string
	for _, m := range reLinkedWiki.FindAllStringSubmatchIndex(lines[line], -1) {
		// m[2], m[3] are the capture group (the text inside [[ ]])
		innerStart := m[2]
		innerEnd := m[3]
		if col >= innerStart && col <= innerEnd {
			targetName = lines[line][innerStart:innerEnd]
			break
		}
	}
	if targetName == "" {
		return nil, nil
	}

	// Find all occurrences of [[targetName]] in the document
	var ranges []protocol.Range
	lower := strings.ToLower(targetName)
	for lineIdx, l := range lines {
		for _, m := range reLinkedWiki.FindAllStringSubmatchIndex(l, -1) {
			inner := l[m[2]:m[3]]
			if strings.ToLower(inner) == lower {
				ranges = append(ranges, protocol.Range{
					Start: protocol.Position{
						Line:      protocol.UInteger(lineIdx),
						Character: protocol.UInteger(m[2]),
					},
					End: protocol.Position{
						Line:      protocol.UInteger(lineIdx),
						Character: protocol.UInteger(m[3]),
					},
				})
			}
		}
	}

	if len(ranges) < 2 {
		return nil, nil // linked editing needs at least 2 ranges
	}

	wordPattern := `[^\]]+`
	return &protocol.LinkedEditingRanges{
		Ranges:      ranges,
		WordPattern: &wordPattern,
	}, nil
}
