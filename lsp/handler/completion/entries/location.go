package entries

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// LocationCompletions provides file path completions for local references.
func LocationCompletions(i []protocol.CompletionItem, query string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	// Future: scan workspace for matching file paths
	_ = query
	return items
}

// AnchorCompletions provides anchor targets (headings) within the current document.
// Wraps HeaderAnchorCompletions for backward compatibility.
func AnchorCompletions(i []protocol.CompletionItem, docContent string, query string) []protocol.CompletionItem {
	return HeaderAnchorCompletions(i, docContent, query)
}

// HeaderCompletions provides heading name completions when typing a heading.
// Triggers when user types `# ` or `## ` etc.
func HeaderCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	headingKind := protocol.CompletionItemKindStruct
	levels := []struct {
		label, insert string
	}{
		{"Heading 1", "# "},
		{"Heading 2", "## "},
		{"Heading 3", "### "},
		{"Heading 4", "#### "},
		{"Heading 5", "##### "},
		{"Heading 6", "###### "},
	}

	for _, l := range levels {
		label := l.label
		insert := l.insert
		detail := "Section heading"
		items = append(items, protocol.CompletionItem{
			Label:            label,
			InsertText:       &insert,
			InsertTextFormat: &TextFormat,
			Kind:             &headingKind,
			Detail:           &detail,
			Documentation:    Documentation(l.label + " — creates a section heading"),
			CommitCharacters: CommitCharacters,
		})
	}

	return items
}

// detectHeaderAnchorTrigger checks if the cursor is inside a markdown link
// that expects an anchor target, e.g. [text](#|) or [text](|).
func DetectHeaderAnchorTrigger(docContent string, line int, col int) (isTrigger bool, query string) {
	lines := strings.Split(docContent, "\n")
	if line >= len(lines) {
		return false, ""
	}
	curr := lines[line]
	if col > len(curr) {
		col = len(curr)
	}
	prefix := curr[:col]

	// Match [text](#... or ](#... or <a href="#...
	if idx := strings.LastIndex(prefix, "(#"); idx >= 0 {
		query = prefix[idx+2:]
		return true, query
	}
	if idx := strings.LastIndex(prefix, "](#"); idx >= 0 {
		query = prefix[idx+4:]
		return true, query
	}

	return false, ""
}
