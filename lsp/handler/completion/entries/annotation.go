package entries

import (
	"os"
	"path/filepath"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var frontmatterFieldKind = protocol.CompletionItemKindProperty

// FrontmatterFieldCompletions provides property name completions for YAML frontmatter.
// Called when cursor is inside a frontmatter block (between --- delimiters at file start).
func FrontmatterFieldCompletions(
	i []protocol.CompletionItem,
	query string,
) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	fields := []struct {
		name, detail, doc string
	}{
		{"title", "Page title", "The title of the document, used in page references and search"},
		{"tags", "Comma-separated tags", "Tags for categorization: tag1, tag2, tag3"},
		{"status", "Content status", "One of: draft, in-progress, review, published, archived"},
		{"date", "Creation date", "ISO 8601 format: YYYY-MM-DD"},
		{"updated", "Last modified date", "ISO 8601 format: YYYY-MM-DD"},
		{"author", "Document author", "Name of the document author"},
		{"project", "Associated project", "Link to project entity"},
		{"due", "Due date", "ISO 8601 format: YYYY-MM-DD"},
		{"priority", "Task priority", "A (highest) through E (lowest)"},
		{"type", "Content type", "One of: note, task, meeting, reference, template"},
		{"description", "Brief description", "Short summary shown in previews and link cards"},
		{"aliases", "Alternative names", "Comma-separated aliases for cross-referencing"},
		{"cssclass", "CSS class for export", "CSS class applied during HTML/PDF export"},
		{"publish", "Publish status", "Whether this page should be published: true/false"},
		{"template", "Template name", "Template used when creating this document"},
		{"version", "Document version", "Semantic version or arbitrary version string"},
	}

	for _, field := range fields {
		if query != "" && !strings.HasPrefix(strings.ToLower(field.name), strings.ToLower(query)) {
			continue
		}
		label := field.name + ": "
		insert := field.name + ": "
		detail := field.detail
		items = append(items, protocol.CompletionItem{
			Label:            label,
			InsertText:       &insert,
			InsertTextFormat: &TextFormat,
			Kind:             &frontmatterFieldKind,
			Detail:           &detail,
			Documentation:    Documentation(field.doc),
			CommitCharacters: CommitCharacters,
			Preselect:        &f,
		})
	}

	return items
}

// FrontmatterValueCompletions provides value completions for known frontmatter keys.
func FrontmatterValueCompletions(
	i []protocol.CompletionItem,
	key string,
	query string,
) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	values := map[string][]string{
		"status":   {"draft", "in-progress", "review", "published", "archived"},
		"priority": {"A", "B", "C", "D", "E"},
		"type":     {"note", "task", "meeting", "reference", "template", "daily"},
		"publish":  {"true", "false"},
	}

	vals, ok := values[strings.ToLower(key)]
	if !ok {
		return items
	}

	for _, v := range vals {
		if query != "" && !strings.HasPrefix(strings.ToLower(v), strings.ToLower(query)) {
			continue
		}
		detail := "Value: " + v
		items = append(items, protocol.CompletionItem{
			Label:            v,
			InsertText:       &v,
			InsertTextFormat: &TextFormat,
			Kind:             &frontmatterFieldKind,
			Detail:           &detail,
			CommitCharacters: CommitCharacters,
		})
	}

	return items
}

// TemplateCompletions provides completions for template names found in .down/templates/.
func TemplateCompletions(i []protocol.CompletionItem, query string, downDir string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	if downDir == "" {
		return items
	}

	tmplDir := filepath.Join(downDir, "templates")
	entries, err := os.ReadDir(tmplDir)
	if err != nil {
		return items
	}

	templateKind := protocol.CompletionItemKindFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		nameNoExt := strings.TrimSuffix(name, ".md")
		if query != "" && !strings.HasPrefix(strings.ToLower(nameNoExt), strings.ToLower(query)) {
			continue
		}
		detail := "Template: " + name
		items = append(items, protocol.CompletionItem{
			Label:            nameNoExt,
			InsertText:       &nameNoExt,
			InsertTextFormat: &TextFormat,
			Kind:             &templateKind,
			Detail:           &detail,
			Documentation:    Documentation("Use template `" + name + "`"),
			CommitCharacters: CommitCharacters,
		})
	}

	// Also offer built-in templates
	for _, bt := range DefaultDocTemplates {
		if query != "" && !strings.HasPrefix(strings.ToLower(bt.Name), strings.ToLower(query)) {
			continue
		}
		insert := bt.Content
		detail := bt.Description + " (built-in)"
		snippetKind := protocol.CompletionItemKindSnippet
		items = append(items, protocol.CompletionItem{
			Label:            bt.Name,
			InsertText:       &insert,
			InsertTextFormat: &SnippetFormat,
			Kind:             &snippetKind,
			Detail:           &detail,
			Documentation:    Documentation(bt.Description),
			CommitCharacters: CommitCharacters,
		})
	}

	return items
}

// HeaderAnchorCompletions provides heading completions when typing an anchor link.
// Triggers when user types `[text](#` or `](#`.
func HeaderAnchorCompletions(
	i []protocol.CompletionItem,
	docContent string,
	query string,
) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	lines := strings.Split(docContent, "\n")
	seen := make(map[string]bool)
	anchorKind := protocol.CompletionItemKindReference

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Extract heading text after the # markers
		heading := strings.TrimLeft(trimmed, "# ")
		heading = strings.TrimSpace(heading)
		if heading == "" || seen[heading] {
			continue
		}
		seen[heading] = true

		// Generate anchor slug: lowercase, spaces/dots to hyphens, strip special chars
		slug := strings.ToLower(heading)
		slug = strings.ReplaceAll(slug, " ", "-")
		slug = strings.ReplaceAll(slug, ".", "-")
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				return r
			}
			return -1
		}, slug)

		if query != "" && !strings.HasPrefix(slug, query) && !strings.Contains(strings.ToLower(heading), strings.ToLower(query)) {
			continue
		}

		detail := "# " + heading
		items = append(items, protocol.CompletionItem{
			Label:            heading,
			InsertText:       &slug,
			InsertTextFormat: &TextFormat,
			Kind:             &anchorKind,
			Detail:           &detail,
			Documentation:    Documentation("Link to `#" + slug + "`"),
			CommitCharacters: CommitCharacters,
			FilterText:       &slug,
		})
	}

	return items
}

// FlagCompletions provides boolean/enum value completions for frontmatter flags.
func FlagCompletions(i []protocol.CompletionItem, key string) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)

	flagKind := protocol.CompletionItemKindEnum
	flags := map[string][]struct{ label, desc string }{
		"auto_index":     {{"true", "Enable automatic indexing"}, {"false", "Disable automatic indexing"}},
		"auto_save":      {{"true", "Enable auto-save"}, {"false", "Disable auto-save"}},
		"show_inlay":     {{"true", "Show inlay hints"}, {"false", "Hide inlay hints"}},
		"spell_check":    {{"true", "Enable spell check"}, {"false", "Disable spell check"}},
		"word_wrap":      {{"true", "Enable word wrap"}, {"false", "Disable word wrap"}},
		"readonly":       {{"true", "Read-only mode"}, {"false", "Editable mode"}},
		"collapsed":      {{"true", "Start collapsed"}, {"false", "Start expanded"}},
		"archived":       {{"true", "Archived"}, {"false", "Active"}},
	}

	opts, ok := flags[strings.ToLower(key)]
	if !ok {
		return items
	}

	for _, o := range opts {
		detail := o.desc
		items = append(items, protocol.CompletionItem{
			Label:            o.label,
			InsertText:       &o.label,
			InsertTextFormat: &TextFormat,
			Kind:             &flagKind,
			Detail:           &detail,
			CommitCharacters: CommitCharacters,
		})
	}

	return items
}
