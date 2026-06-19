package entries

import (
	"github.com/clpi/down/internal/data"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var WorkspaceCompletions = func(i []protocol.CompletionItem) []protocol.CompletionItem {
	mode := protocol.InsertTextModeAsIs
	wsl := data.Workspaces()
	print(wsl)
	kind := protocol.CompletionItemKindFolder
	items := i
	for _, w := range wsl {
		uri := w.URI
		name := w.Name
		items = append(items, protocol.CompletionItem{
			Label:            name,
			Documentation:    "# " + name + "\n\n" + uri,
			InsertText:       &uri,
			CommitCharacters: []string{"/", "[", "(", "<"},
			InsertTextMode:   &mode,
			Detail:           &uri,
			Kind:             &kind,
		})

	}
	return items
}

var FileCompletions = func(i []protocol.CompletionItem) []protocol.CompletionItem {
	mode := protocol.InsertTextModeAsIs
	file := "file"
	desc := "file completion"
	items := i
	kind := protocol.CompletionItemKindFile
	items = append(items, protocol.CompletionItem{
		Label:            "file",
		Documentation:    "# File completion",
		InsertText:       &file,
		CommitCharacters: []string{"/", "[", "(", "<"},
		InsertTextMode:   &mode,
		Detail:           &desc,
		Kind:             &kind,
	})
	return items
}
