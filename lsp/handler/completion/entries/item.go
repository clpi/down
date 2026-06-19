package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

var (
	CommitCharacters = []string{" ", "\t", "\n"}
	Tags             = []protocol.CompletionItemTag{
		protocol.CompletionItemTagDeprecated,
	}
	Documentation = func(d string) protocol.MarkupContent {
		return protocol.MarkupContent{
			Value: d,
			Kind:  protocol.MarkupKindMarkdown,
		}
	}
	InsertAdjust  = protocol.InsertTextModeAdjustIndentation
	SnippetFormat = protocol.InsertTextFormatSnippet
	TextFormat    = protocol.InsertTextFormatPlainText
	SnippetKind   = protocol.CompletionItemKindSnippet
	SnippetItem   = func(
		label string,
		insert string,
		detail string,
		doc string,
	) protocol.CompletionItem {
		return protocol.CompletionItem{
			Label:               label,
			InsertText:          &insert,
			TextEdit:            []protocol.TextEdit{},
			Data:                nil,
			AdditionalTextEdits: []protocol.TextEdit{},
			CommitCharacters:    CommitCharacters,
			InsertTextFormat:    &SnippetFormat,
			InsertTextMode:      &InsertAdjust,
			Command:             nil,
			Kind:                &SnippetKind,
			Detail:              &detail,
			Deprecated:          &f,
			Tags:                Tags,
			Documentation:       Documentation(doc),
			Preselect:           &t,
		}
	}
	CompletionItem = func(
		label string,
		insert string,
		detail string,
		doc string,
		kind protocol.CompletionItemKind,
	) protocol.CompletionItem {
		return protocol.CompletionItem{
			Label:               label,
			InsertText:          &insert,
			TextEdit:            []protocol.TextEdit{},
			Data:                nil,
			AdditionalTextEdits: []protocol.TextEdit{},
			CommitCharacters:    CommitCharacters,
			InsertTextFormat:    &TextFormat,
			InsertTextMode:      &InsertAdjust,
			Command:             nil,
			Kind:                &kind,
			Detail:              &detail,
			Deprecated:          &f,
			Tags:                Tags,
			Documentation:       Documentation(doc),
			Preselect:           &t,
		}
	}
)

var (
	Completion = func() protocol.CompletionItem {
		i := protocol.CompletionItem{}
		return i
	}
	Completions = func(i []protocol.CompletionItem) []protocol.CompletionItem {
		items := append([]protocol.CompletionItem{})
		for _, c := range i {
			items = append(items, c)
		}
		return items
	}
)
