package handler

import (
	"fmt"
	"strings"

	"github.com/clpi/down/lsp/knowledge"
)

// InlayHintData represents an inlay hint (custom implementation for pre-3.17 protocol).
// This can be exposed via custom commands until the protocol library supports 3.17.
type InlayHintData struct {
	Line      int    `json:"line"`
	Character int    `json:"character"`
	Label     string `json:"label"`
	Kind      string `json:"kind"` // "type" or "parameter"
}

// ComputeInlayHints generates inlay hints for a document range.
// This is available via the down.inlayhints command until protocol 3.17 is supported.
func (s *State) ComputeInlayHints(uri string, startLine, endLine int) []InlayHintData {
	text, ok := s.Documents[uri]
	if !ok {
		return nil
	}

	lines := strings.Split(text, "\n")
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	var hints []InlayHintData
	inCode := false
	taskCount := 0
	doneCount := 0

	for i := startLine; i <= endLine; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}

		// Heading hints - show word count for section
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
				wordCount := countSectionWords(lines, i, level)
				if wordCount > 0 {
					hints = append(hints, InlayHintData{
						Line:      i,
						Character: len(line),
						Label:     fmt.Sprintf(" (%d words)", wordCount),
						Kind:      "type",
					})
				}
			}
		}

		// Task progress
		if strings.Contains(trimmed, "- [ ]") {
			taskCount++
		} else if strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
			taskCount++
			doneCount++
		}

		// Wiki link mention counts
		if s.Graph != nil {
			for _, m := range reLinkWiki.FindAllStringSubmatch(line, -1) {
				parts := strings.SplitN(m[1], "|", 2)
				target := strings.TrimSpace(parts[0])
				results := s.Graph.Search(target)
				for _, ent := range results {
					if strings.EqualFold(ent.Name, target) && ent.Mentions > 1 {
						col := strings.Index(line, m[0]) + len(m[0])
						hints = append(hints, InlayHintData{
							Line:      i,
							Character: col,
							Label:     fmt.Sprintf(" x%d", ent.Mentions),
							Kind:      "parameter",
						})
						break
					}
				}
			}
		}
	}

	// Document-level task progress
	if taskCount > 0 {
		hints = append(hints, InlayHintData{
			Line:      startLine,
			Character: len(lines[startLine]),
			Label:     fmt.Sprintf(" [%d/%d tasks]", doneCount, taskCount),
			Kind:      "type",
		})
	}

	// Knowledge graph entity count
	if s.Graph != nil {
		entities := s.Graph.EntitiesByDocument(uri)
		if len(entities) > 0 {
			entityTypes := make(map[knowledge.EntityKind]int)
			for _, ent := range entities {
				entityTypes[ent.Kind]++
			}
			var parts []string
			for kind, count := range entityTypes {
				parts = append(parts, fmt.Sprintf("%s:%d", kind, count))
			}
			if len(parts) > 0 && startLine == 0 {
				hints = append(hints, InlayHintData{
					Line:      0,
					Character: len(lines[0]),
					Label:     fmt.Sprintf(" {%s}", strings.Join(parts, ", ")),
					Kind:      "type",
				})
			}
		}
	}

	return hints
}

func countSectionWords(lines []string, headingLine int, headingLevel int) int {
	count := 0
	for i := headingLine + 1; i < len(lines); i++ {
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
			if level <= headingLevel {
				break
			}
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "```") {
			words := strings.Fields(trimmed)
			count += len(words)
		}
	}
	return count
}
