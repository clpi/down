package entries

import (
	"strings"

	"github.com/clpi/down/lsp/knowledge"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func WikiLinkCompletions(items []protocol.CompletionItem, graph *knowledge.Graph, query string) []protocol.CompletionItem {
	if graph == nil {
		return items
	}

	docs := graph.EntitiesByKind(knowledge.KindDocument)
	kind := protocol.CompletionItemKindReference

	for _, d := range docs {
		name := d.Name
		if strings.Contains(name, "/") || strings.Contains(name, "\\") {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			continue
		}

		insertText := name
		label := "[[" + name + "]]"
		detail := "Page"

		items = append(items, protocol.CompletionItem{
			Label:      label,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &insertText,
			FilterText: &name,
		})
	}

	return items
}
func WorkspaceFileCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
func FileLinkCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
func ImageCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
