package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

func LocationCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
func AnchorCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
func HeaderCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	return items
}
