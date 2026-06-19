package handler

import (
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	reTrailingWS    = regexp.MustCompile(`[ \t]+$`)
	reMultipleBlank = regexp.MustCompile(`\n{3,}`)
	reHeadingSpace  = regexp.MustCompile(`^(#{1,6})([^ #\n])`)
	reListIndent    = regexp.MustCompile(`^(\s*)([*+-]|\d+\.) {2,}`)
)

// Format implements textDocument/formatting.
// It normalizes whitespace, heading spacing, and list formatting.
func (s *State) Format(_ *glsp.Context, p *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	formatted := formatMarkdown(text, p.Options)
	if formatted == text {
		return nil, nil // no changes
	}

	// Replace the entire document
	lines := strings.Split(text, "\n")
	lastLine := len(lines) - 1
	lastChar := len(lines[lastLine])

	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End: protocol.Position{
					Line:      protocol.UInteger(lastLine),
					Character: protocol.UInteger(lastChar),
				},
			},
			NewText: formatted,
		},
	}, nil
}

// RangeFormat implements textDocument/rangeFormatting.
// Formats only the selected range of the document.
func (s *State) RangeFormat(_ *glsp.Context, p *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	startLine := int(p.Range.Start.Line)
	endLine := int(p.Range.End.Line)

	if startLine >= len(lines) {
		return nil, nil
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// Extract the range text
	rangeLines := lines[startLine : endLine+1]
	rangeText := strings.Join(rangeLines, "\n")

	// Format just this range
	formatted := formatMarkdownRange(rangeText, p.Options)
	if formatted == rangeText {
		return nil, nil
	}

	// Compute end character of the last line in range
	endChar := len(lines[endLine])

	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(startLine), Character: 0},
				End:   protocol.Position{Line: protocol.UInteger(endLine), Character: protocol.UInteger(endChar)},
			},
			NewText: formatted,
		},
	}, nil
}

// formatMarkdownRange formats a subset of markdown lines.
// Similar to formatMarkdown but without document-level concerns (frontmatter, final newline).
func formatMarkdownRange(text string, opts protocol.FormattingOptions) string {
	lines := strings.Split(text, "\n")
	var result []string

	inCodeBlock := false
	prevBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track code blocks — don't format inside them
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			result = append(result, line)
			prevBlank = false
			continue
		}
		if inCodeBlock {
			result = append(result, line)
			prevBlank = false
			continue
		}

		// Remove trailing whitespace (preserve intentional 2-space line breaks)
		stripped := strings.TrimRight(line, " \t")
		if len(line) > len(stripped) && len(line)-len(stripped) >= 2 {
			line = stripped + "  "
		} else {
			line = stripped
		}

		// Ensure space after heading marker
		line = reHeadingSpace.ReplaceAllString(line, "$1 $2")

		// Normalize list item spacing
		if m := reListIndent.FindStringSubmatch(line); m != nil {
			rest := strings.TrimSpace(line[len(m[0]):])
			line = m[1] + m[2] + " " + rest
		}

		// Collapse multiple blank lines
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && prevBlank {
			continue
		}

		// Ensure blank line before headings
		if strings.HasPrefix(trimmed, "#") && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
			result = append(result, "")
		}

		result = append(result, line)
		prevBlank = isBlank
	}

	return strings.Join(result, "\n")
}

func formatMarkdown(text string, opts protocol.FormattingOptions) string {
	lines := strings.Split(text, "\n")
	var result []string

	inCodeBlock := false
	inFrontmatter := false
	prevBlank := false

	for i, line := range lines {
		// Track frontmatter
		if i == 0 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			result = append(result, line)
			prevBlank = false
			continue
		}
		if inFrontmatter {
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
			}
			result = append(result, line)
			prevBlank = false
			continue
		}

		// Track code blocks — don't format inside them
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			result = append(result, line)
			prevBlank = false
			continue
		}
		if inCodeBlock {
			result = append(result, line)
			prevBlank = false
			continue
		}

		// Remove trailing whitespace. Preserve intentional line breaks (2 trailing spaces).
		stripped := strings.TrimRight(line, " \t")
		if len(line) > len(stripped) && len(line)-len(stripped) >= 2 {
			// Markdown line break: normalize to exactly 2 trailing spaces
			line = stripped + "  "
		} else {
			line = stripped
		}

		// Ensure space after heading marker: ##Foo → ## Foo
		line = reHeadingSpace.ReplaceAllString(line, "$1 $2")

		// Normalize list item spacing: -  item → - item
		if m := reListIndent.FindStringSubmatch(line); m != nil {
			rest := strings.TrimSpace(line[len(m[0]):])
			line = m[1] + m[2] + " " + rest
		}

		// Collapse multiple blank lines
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && prevBlank {
			continue // skip extra blank line
		}

		// Ensure blank line before headings (unless at start)
		if strings.HasPrefix(trimmed, "#") && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
			result = append(result, "")
		}

		result = append(result, line)
		prevBlank = isBlank
	}

	// Ensure file ends with a single newline
	out := strings.Join(result, "\n")
	out = strings.TrimRight(out, "\n") + "\n"
	return out
}
