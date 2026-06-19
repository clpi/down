package handler

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// FoldingRange implements textDocument/foldingRange.
// Returns folding ranges for headings, code blocks, frontmatter, and lists.
func (s *State) FoldingRange(_ *glsp.Context, p *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	var ranges []protocol.FoldingRange

	// Track headings for section folding
	type headingInfo struct {
		line  int
		level int
	}
	var headings []headingInfo

	inCode := false
	codeStart := -1
	inFrontmatter := false
	fmStart := -1

	regionKind := string(protocol.FoldingRangeKindRegion)
	commentKind := string(protocol.FoldingRangeKindComment)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Frontmatter
		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			fmStart = i
			continue
		}
		if inFrontmatter && trimmed == "---" {
			inFrontmatter = false
			if i > fmStart+1 {
				ranges = append(ranges, protocol.FoldingRange{
					StartLine:      protocol.UInteger(fmStart),
					EndLine:        protocol.UInteger(i),
					Kind:           &commentKind,
				})
			}
			continue
		}

		// Code blocks
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if !inCode {
				inCode = true
				codeStart = i
			} else {
				inCode = false
				if i > codeStart+1 {
					ranges = append(ranges, protocol.FoldingRange{
						StartLine:      protocol.UInteger(codeStart),
						EndLine:        protocol.UInteger(i),
						Kind:           &regionKind,
					})
				}
			}
			continue
		}

		if inCode {
			continue
		}

		// Headings
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for _, ch := range trimmed {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			if level >= 1 && level <= 6 {
				headings = append(headings, headingInfo{line: i, level: level})
			}
		}
	}

	// Build heading folding ranges
	for idx, h := range headings {
		endLine := len(lines) - 1
		for j := idx + 1; j < len(headings); j++ {
			if headings[j].level <= h.level {
				endLine = headings[j].line - 1
				break
			}
		}
		// Remove trailing blank lines
		for endLine > h.line && strings.TrimSpace(lines[endLine]) == "" {
			endLine--
		}
		if endLine > h.line {
			ranges = append(ranges, protocol.FoldingRange{
				StartLine:      protocol.UInteger(h.line),
				EndLine:        protocol.UInteger(endLine),
				Kind:           &regionKind,
			})
		}
	}

	return ranges, nil
}
