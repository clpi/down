package handler

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// OnTypeFormatting implements textDocument/onTypeFormatting.
// Provides auto-continuation for markdown constructs when pressing Enter:
// - Bulleted lists (-, *, +)
// - Numbered lists (1., 2., etc.)
// - Task lists (- [ ])
// - Blockquotes (>)
// - Callout continuations (> )
//
// Also handles smart behavior:
// - Empty list item on Enter → remove the marker (end the list)
// - Increment numbered list items
func (s *State) OnTypeFormatting(_ *glsp.Context, p *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	lineIdx := int(p.Position.Line)

	// We need the PREVIOUS line (the one before the newly created line)
	prevLineIdx := lineIdx - 1
	if prevLineIdx < 0 || prevLineIdx >= len(lines) {
		return nil, nil
	}
	prevLine := lines[prevLineIdx]

	// Only trigger on Enter (newline character)
	if p.Ch != "\n" {
		return nil, nil
	}

	// Detect the construct on the previous line
	indent, marker, content := parseListLine(prevLine)

	if marker == "" {
		// Check for blockquote
		trimmed := strings.TrimSpace(prevLine)
		if strings.HasPrefix(trimmed, "> ") {
			// If the blockquote line has content, continue it
			quoteContent := strings.TrimSpace(trimmed[2:])
			if quoteContent != "" {
				// Continue the blockquote
				insertText := indent + "> "
				return []protocol.TextEdit{
					{
						Range: protocol.Range{
							Start: protocol.Position{Line: protocol.UInteger(lineIdx), Character: 0},
							End:   protocol.Position{Line: protocol.UInteger(lineIdx), Character: p.Position.Character},
						},
						NewText: insertText,
					},
				}, nil
			}
			// Empty blockquote → end it
			return []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(prevLineIdx), Character: 0},
						End:   protocol.Position{Line: protocol.UInteger(lineIdx), Character: p.Position.Character},
					},
					NewText: "\n",
				},
			}, nil
		}
		return nil, nil
	}

	// If the previous line's list item was empty (just the marker), remove it and end the list
	if strings.TrimSpace(content) == "" {
		return []protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(prevLineIdx), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(lineIdx), Character: p.Position.Character},
				},
				NewText: "\n",
			},
		}, nil
	}

	// Continue the list with appropriate marker
	nextMarker := nextListMarker(marker)
	insertText := indent + nextMarker

	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(lineIdx), Character: 0},
				End:   protocol.Position{Line: protocol.UInteger(lineIdx), Character: p.Position.Character},
			},
			NewText: insertText,
		},
	}, nil
}

// parseListLine extracts indent, marker, and content from a list line.
// Returns ("", "", "") if the line is not a list item.
func parseListLine(line string) (indent, marker, content string) {
	// Find leading whitespace
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	indent = line[:i]
	rest := line[i:]

	// Task list: - [ ] or - [x]
	if strings.HasPrefix(rest, "- [ ] ") {
		return indent, "- [ ] ", rest[6:]
	}
	if strings.HasPrefix(rest, "- [x] ") || strings.HasPrefix(rest, "- [X] ") {
		return indent, "- [ ] ", rest[6:] // next task is unchecked
	}
	if strings.HasPrefix(rest, "* [ ] ") {
		return indent, "* [ ] ", rest[6:]
	}
	if strings.HasPrefix(rest, "* [x] ") || strings.HasPrefix(rest, "* [X] ") {
		return indent, "* [ ] ", rest[6:]
	}

	// Bulleted list: -, *, +
	if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
		return indent, string(rest[0]) + " ", rest[2:]
	}

	// Numbered list: 1. 2. etc.
	j := 0
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		j++
	}
	if j > 0 && j < len(rest) {
		if rest[j] == '.' && j+1 < len(rest) && rest[j+1] == ' ' {
			return indent, rest[:j+2], rest[j+2:]
		}
		if rest[j] == ')' && j+1 < len(rest) && rest[j+1] == ' ' {
			return indent, rest[:j+2], rest[j+2:]
		}
	}

	return "", "", ""
}

// nextListMarker returns the next marker for list continuation.
// For numbered lists, it increments the number.
func nextListMarker(marker string) string {
	// Task lists and bullet lists stay the same
	if strings.HasPrefix(marker, "- [ ]") || strings.HasPrefix(marker, "* [ ]") {
		return marker
	}
	if len(marker) == 2 && (marker[0] == '-' || marker[0] == '*' || marker[0] == '+') {
		return marker
	}

	// Numbered lists: extract number and increment
	j := 0
	for j < len(marker) && marker[j] >= '0' && marker[j] <= '9' {
		j++
	}
	if j > 0 {
		num := 0
		for _, ch := range marker[:j] {
			num = num*10 + int(ch-'0')
		}
		num++
		suffix := marker[j:] // ". " or ") "
		return intToStr(num) + suffix
	}

	return marker
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
