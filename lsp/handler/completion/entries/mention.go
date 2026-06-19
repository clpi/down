package entries

import (
	"strings"

	"github.com/clpi/down/lsp/knowledge"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func MentionCompletions(items []protocol.CompletionItem, graph *knowledge.Graph, query string) []protocol.CompletionItem {
	if graph == nil {
		return items
	}

	people := graph.EntitiesByKind(knowledge.KindPerson)
	kind := protocol.CompletionItemKindKeyword

	for _, p := range people {
		name := p.Name
		if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			continue
		}

		insertText := name
		label := "@" + name
		detail := "Person"

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
