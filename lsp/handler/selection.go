package handler

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// SelectionRange implements textDocument/selectionRange.
// Provides smart selection ranges for markdown elements (word → line → paragraph → section → document).
func (s *State) SelectionRange(_ *glsp.Context, p *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	var result []protocol.SelectionRange

	for _, pos := range p.Positions {
		lineIdx := int(pos.Line)
		col := int(pos.Character)
		if lineIdx >= len(lines) {
			result = append(result, protocol.SelectionRange{
				Range: protocol.Range{Start: pos, End: pos},
			})
			continue
		}
		line := lines[lineIdx]
		if col > len(line) {
			col = len(line)
		}

		// Level 1: Word
		wordStart, wordEnd := col, col
		for wordStart > 0 && isWordChar(line[wordStart-1]) {
			wordStart--
		}
		for wordEnd < len(line) && isWordChar(line[wordEnd]) {
			wordEnd++
		}
		wordRange := protocol.Range{
			Start: protocol.Position{Line: pos.Line, Character: protocol.UInteger(wordStart)},
			End:   protocol.Position{Line: pos.Line, Character: protocol.UInteger(wordEnd)},
		}

		// Level 2: Line
		lineRange := protocol.Range{
			Start: protocol.Position{Line: pos.Line, Character: 0},
			End:   protocol.Position{Line: pos.Line, Character: protocol.UInteger(len(line))},
		}

		// Level 3: Paragraph (contiguous non-blank lines)
		paraStart := lineIdx
		for paraStart > 0 && strings.TrimSpace(lines[paraStart-1]) != "" {
			paraStart--
		}
		paraEnd := lineIdx
		for paraEnd < len(lines)-1 && strings.TrimSpace(lines[paraEnd+1]) != "" {
			paraEnd++
		}
		paraRange := protocol.Range{
			Start: protocol.Position{Line: protocol.UInteger(paraStart), Character: 0},
			End:   protocol.Position{Line: protocol.UInteger(paraEnd), Character: protocol.UInteger(len(lines[paraEnd]))},
		}

		// Level 4: Section (from heading to next heading of same or higher level)
		sectionStart := lineIdx
		sectionLevel := 0
		for i := lineIdx; i >= 0; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "#") {
				level := 0
				for _, ch := range trimmed {
					if ch == '#' {
						level++
					} else {
						break
					}
				}
				sectionStart = i
				sectionLevel = level
				break
			}
		}
		sectionEnd := len(lines) - 1
		if sectionLevel > 0 {
			for i := lineIdx + 1; i < len(lines); i++ {
				trimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(trimmed, "#") {
					level := 0
					for _, ch := range trimmed {
						if ch == '#' {
							level++
						} else {
							break
						}
					}
					if level <= sectionLevel {
						sectionEnd = i - 1
						break
					}
				}
			}
		}
		sectionRange := protocol.Range{
			Start: protocol.Position{Line: protocol.UInteger(sectionStart), Character: 0},
			End:   protocol.Position{Line: protocol.UInteger(sectionEnd), Character: protocol.UInteger(len(lines[sectionEnd]))},
		}

		// Level 5: Document
		docRange := protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: protocol.UInteger(len(lines) - 1), Character: protocol.UInteger(len(lines[len(lines)-1]))},
		}

		// Build nested selection ranges (inner → outer)
		docSel := protocol.SelectionRange{Range: docRange}
		sectionSel := protocol.SelectionRange{Range: sectionRange, Parent: &docSel}
		paraSel := protocol.SelectionRange{Range: paraRange, Parent: &sectionSel}
		lineSel := protocol.SelectionRange{Range: lineRange, Parent: &paraSel}
		wordSel := protocol.SelectionRange{Range: wordRange, Parent: &lineSel}

		result = append(result, wordSel)
	}
	return result, nil
}

