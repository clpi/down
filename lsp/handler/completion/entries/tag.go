package entries

import (
	"strings"

	"github.com/clpi/down/lsp/knowledge"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TagCompletions(items []protocol.CompletionItem, graph *knowledge.Graph, query string) []protocol.CompletionItem {
	if graph == nil {
		return items
	}

	tags := graph.EntitiesByKind(knowledge.KindTag)
	kind := protocol.CompletionItemKindKeyword

	for _, t := range tags {
		name := t.Name
		if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			continue
		}

		insertText := name
		label := "#" + name
		detail := "Tag"

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
