package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

func (ew *Workspace) SnippetCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := i
	for s, sn := range Snippets {
		kind := protocol.CompletionItemKindSnippet
		items = append(items, protocol.CompletionItem{
			Tags: []protocol.CompletionItemTag{
				protocol.CompletionItemTagDeprecated,
			},
			Label:            s,
			Kind:             &kind,
			CommitCharacters: CommitCharacters,
			Preselect:        &t,
			Documentation: &protocol.MarkupContent{
				Value: "# Snippets\n\n## Snippet\n_ _ _\n### Snippet: " + sn.Description + "\n---\n" + sn.Body,
				Kind:  protocol.MarkupKindMarkdown,
			},
			Detail:     &sn.Description,
			InsertText: &sn.Body,
		})

	}
	return items
}

// var
func SnippetCompletions(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := i
	for s, sn := range Snippets {
		kind := protocol.CompletionItemKindSnippet
		items = append(items, protocol.CompletionItem{
			Label:            s,
			CommitCharacters: CommitCharacters,
			Kind:             &kind,
			Preselect:        &t,
			Documentation: &protocol.MarkupContent{
				Value: "# Snippets\n\n## Snippet\n_ _ _\n### Snippet: " + sn.Description + "\n---\n" + sn.Body,
				Kind:  protocol.MarkupKindMarkdown,
			},
			Detail:     &sn.Description,
			InsertText: &sn.Body,
		})

	}
	return items
}
