package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

func RefCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
