package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

const ()

func TemplateCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
