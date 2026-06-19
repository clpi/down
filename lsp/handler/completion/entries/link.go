package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

func WikiLinkCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
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
