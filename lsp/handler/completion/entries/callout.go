package entries

import (
	"strings"
	"time"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// CalloutCompletions provides completions for callout types triggered by "> [!".
func CalloutCompletions(i []protocol.CompletionItem, query string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	callouts := []struct {
		name, emoji, desc string
	}{
		{"NOTE", "📝", "General information or notes"},
		{"TIP", "💡", "Helpful tips and suggestions"},
		{"IMPORTANT", "⚠️", "Important information to note"},
		{"WARNING", "🚨", "Warnings and cautions"},
		{"CAUTION", "🔥", "Critical cautions"},
		{"DANGER", "💀", "Dangerous or destructive actions"},
		{"INFO", "ℹ️", "Informational content"},
		{"TODO", "✅", "Action items or todos"},
		{"DONE", "✔️", "Completed items"},
		{"FAQ", "❓", "Frequently asked questions"},
		{"SUCCESS", "🎉", "Success messages or achievements"},
		{"FAILURE", "❌", "Failure messages or errors"},
		{"BUG", "🐛", "Known bugs or issues"},
		{"EXAMPLE", "📖", "Code examples or demonstrations"},
		{"QUOTE", "💬", "Quoted content"},
		{"ABSTRACT", "📄", "Summary or abstract"},
	}

	calloutKind := protocol.CompletionItemKindKeyword
	for _, c := range callouts {
		if query != "" && !strings.HasPrefix(strings.ToLower(c.name), strings.ToLower(query)) {
			continue
		}
		label := c.name
		insert := c.name + "]"
		detail := c.emoji + " " + c.desc
		items = append(items, protocol.CompletionItem{
			Label:            label,
			InsertText:       &insert,
			InsertTextFormat: &TextFormat,
			Kind:             &calloutKind,
			Detail:           &detail,
			Documentation:    Documentation(c.desc),
			CommitCharacters: CommitCharacters,
		})
	}
	return items
}

// DateCompletions provides natural language date completions.
func DateCompletions(i []protocol.CompletionItem, query string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	now := time.Now()
	dateKind := protocol.CompletionItemKindValue

	shortcuts := map[string]string{
		"today":     now.Format("2006-01-02"),
		"tomorrow":  now.AddDate(0, 0, 1).Format("2006-01-02"),
		"yesterday": now.AddDate(0, 0, -1).Format("2006-01-02"),
		"next week": now.AddDate(0, 0, 7).Format("2006-01-02"),
		"last week": now.AddDate(0, 0, -7).Format("2006-01-02"),
		"next month": now.AddDate(0, 1, 0).Format("2006-01-02"),
		"last month": now.AddDate(0, -1, 0).Format("2006-01-02"),
	}

	for label, value := range shortcuts {
		if query != "" && !strings.HasPrefix(strings.ToLower(label), strings.ToLower(query)) {
			continue
		}
		detail := value
		items = append(items, protocol.CompletionItem{
			Label:            label,
			InsertText:       &value,
			InsertTextFormat: &TextFormat,
			Kind:             &dateKind,
			Detail:           &detail,
			Documentation:    Documentation("Insert date: " + value),
			CommitCharacters: CommitCharacters,
		})
	}
	return items
}

// PropertyValueCompletions provides completions for known property values from the workspace.
func PropertyValueCompletions(i []protocol.CompletionItem, key, query string, existingValues []string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	if len(existingValues) == 0 {
		return items
	}

	propKind := protocol.CompletionItemKindValue
	seen := make(map[string]bool)
	for _, v := range existingValues {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		if query != "" && !strings.HasPrefix(strings.ToLower(v), strings.ToLower(query)) {
			continue
		}
		detail := key + ": " + v
		items = append(items, protocol.CompletionItem{
			Label:            v,
			InsertText:       &v,
			InsertTextFormat: &TextFormat,
			Kind:             &propKind,
			Detail:           &detail,
			CommitCharacters: CommitCharacters,
		})
	}
	return items
}

// TableColumnCompletions provides column name completions for markdown table headers.
func TableColumnCompletions(i []protocol.CompletionItem, query string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	common := []string{"Status", "Date", "Priority", "Assignee", "Tags", "Notes", "Due", "Project", "Type", "Category"}
	colKind := protocol.CompletionItemKindField
	for _, c := range common {
		if query != "" && !strings.HasPrefix(strings.ToLower(c), strings.ToLower(query)) {
			continue
		}
		detail := "Column: " + c
		items = append(items, protocol.CompletionItem{
			Label:            c,
			InsertText:       &c,
			InsertTextFormat: &TextFormat,
			Kind:             &colKind,
			Detail:           &detail,
			CommitCharacters: CommitCharacters,
		})
	}
	return items
}
