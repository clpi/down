package entries

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// StatusCompletions provides status value completions (draft, in-progress, etc.).
func StatusCompletions(i []protocol.CompletionItem, query string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	statuses := []string{"draft", "in-progress", "review", "published", "archived", "done", "cancelled"}

	enumKind := protocol.CompletionItemKindEnum
	for _, s := range statuses {
		if query != "" && !strings.HasPrefix(s, query) {
			continue
		}
		detail := "Status: " + s
		items = append(items, protocol.CompletionItem{
			Label:            s,
			InsertText:       &s,
			InsertTextFormat: &TextFormat,
			Kind:             &enumKind,
			Detail:           &detail,
			CommitCharacters: CommitCharacters,
		})
	}
	return items
}

// YAMLValueCompletions provides completions for common YAML boolean/enum values.
func YAMLValueCompletions(i []protocol.CompletionItem, key string) []protocol.CompletionItem {
	return FlagCompletions(i, key)
}
