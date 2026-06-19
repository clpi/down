package entries

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func EmojiCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := i
	for w, e := range Emojis {
		ec := e
		kind := protocol.CompletionItemKindConstant
		items = append(items, protocol.CompletionItem{
			Label:            w,
			Kind:             &kind,
			Deprecated:       &f,
			Data:             nil,
			CommitCharacters: CommitCharacters,
			InsertTextMode:   &InsertAdjust,
			InsertTextFormat: &TextFormat,
			Command:          nil,
			// TextEdit:            []protocol.TextEdit{},
			// AdditionalTextEdits: []protocol.TextEdit{},
			Preselect: &t,
			Tags:      []protocol.CompletionItemTag{
				// protocol.CompletionItemTagDeprecated,
			},
			Documentation: &protocol.MarkupContent{
				Value: "# Emoji\n\n## Emoji\n_ _ _\n### Emoji: " + ec + "\n---\n" + ec,
				Kind:  protocol.MarkupKindMarkdown,
			},
			Detail:     &ec,
			InsertText: &ec,
		})
	}
	return items
}
