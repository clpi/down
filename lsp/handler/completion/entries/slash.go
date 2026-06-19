package entries

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// SlashCommand defines a Notion-style /command with its insert template.
type SlashCommand struct {
	Label       string
	Detail      string
	Description string
	InsertText  string
	Category    string
	SortOrder   string
}

// SlashCommands contains all available slash commands grouped by category.
var SlashCommands = []SlashCommand{
	// Basic blocks
	{Label: "/text", Detail: "Plain text", Description: "Insert a plain text block", InsertText: "", Category: "Basic", SortOrder: "01a"},
	{Label: "/heading1", Detail: "Heading 1", Description: "Large section heading", InsertText: "# ${1:Heading}\n", Category: "Basic", SortOrder: "01b"},
	{Label: "/heading2", Detail: "Heading 2", Description: "Medium section heading", InsertText: "## ${1:Heading}\n", Category: "Basic", SortOrder: "01c"},
	{Label: "/heading3", Detail: "Heading 3", Description: "Small section heading", InsertText: "### ${1:Heading}\n", Category: "Basic", SortOrder: "01d"},
	{Label: "/bullet", Detail: "Bulleted list", Description: "Create a bulleted list", InsertText: "- ${1:Item}\n- ${2:Item}\n- ${3:Item}\n", Category: "Basic", SortOrder: "01e"},
	{Label: "/numbered", Detail: "Numbered list", Description: "Create a numbered list", InsertText: "1. ${1:Item}\n2. ${2:Item}\n3. ${3:Item}\n", Category: "Basic", SortOrder: "01f"},
	{Label: "/todo", Detail: "To-do list", Description: "Task list with checkboxes", InsertText: "- [ ] ${1:Task}\n- [ ] ${2:Task}\n- [ ] ${3:Task}\n", Category: "Basic", SortOrder: "01g"},
	{Label: "/toggle", Detail: "Toggle list", Description: "Collapsible toggle section", InsertText: "<details>\n<summary>${1:Toggle title}</summary>\n\n${2:Hidden content}\n\n</details>\n", Category: "Basic", SortOrder: "01h"},
	{Label: "/quote", Detail: "Quote", Description: "Insert a blockquote", InsertText: "> ${1:Quote text}\n>\n> — ${2:Attribution}\n", Category: "Basic", SortOrder: "01i"},
	{Label: "/divider", Detail: "Divider", Description: "Horizontal rule separator", InsertText: "\n---\n\n", Category: "Basic", SortOrder: "01j"},
	{Label: "/link", Detail: "Link to page", Description: "Link to another document", InsertText: "[[${1:page}]]", Category: "Basic", SortOrder: "01k"},

	// Media & Embeds
	{Label: "/image", Detail: "Image", Description: "Insert an image", InsertText: "![${1:alt text}](${2:url})\n", Category: "Media", SortOrder: "02a"},
	{Label: "/code", Detail: "Code block", Description: "Insert a fenced code block", InsertText: "```${1:language}\n${2:code}\n```\n", Category: "Media", SortOrder: "02b"},
	{Label: "/math", Detail: "Math block", Description: "Insert a LaTeX math block", InsertText: "$$\n${1:formula}\n$$\n", Category: "Media", SortOrder: "02c"},
	{Label: "/embed", Detail: "Embed", Description: "Embed a file or page", InsertText: "![[${1:file}]]\n", Category: "Media", SortOrder: "02d"},
	{Label: "/video", Detail: "Video embed", Description: "Embed a video link", InsertText: "[![${1:Video}](${2:thumbnail_url})](${3:video_url})\n", Category: "Media", SortOrder: "02e"},
	{Label: "/file", Detail: "File attachment", Description: "Reference a file", InsertText: "[📎 ${1:filename}](${2:path})\n", Category: "Media", SortOrder: "02f"},
	{Label: "/bookmark", Detail: "Web bookmark", Description: "Save a web bookmark", InsertText: "> [!bookmark] [${1:Title}](${2:URL})\n", Category: "Media", SortOrder: "02g"},

	// Callouts
	{Label: "/callout", Detail: "Callout", Description: "Highlighted callout block", InsertText: "> [!${1|note,tip,warning,danger,info|}]\n> ${2:Callout content}\n", Category: "Callout", SortOrder: "03a"},
	{Label: "/note", Detail: "Note callout", Description: "Informational note", InsertText: "> [!note]\n> ${1:Note content}\n", Category: "Callout", SortOrder: "03b"},
	{Label: "/tip", Detail: "Tip callout", Description: "Helpful tip", InsertText: "> [!tip]\n> ${1:Tip content}\n", Category: "Callout", SortOrder: "03c"},
	{Label: "/warning", Detail: "Warning callout", Description: "Warning notice", InsertText: "> [!warning]\n> ${1:Warning content}\n", Category: "Callout", SortOrder: "03d"},
	{Label: "/danger", Detail: "Danger callout", Description: "Danger/error alert", InsertText: "> [!danger]\n> ${1:Danger content}\n", Category: "Callout", SortOrder: "03e"},
	{Label: "/info", Detail: "Info callout", Description: "Information block", InsertText: "> [!info]\n> ${1:Information}\n", Category: "Callout", SortOrder: "03f"},

	// Advanced blocks
	{Label: "/table", Detail: "Table", Description: "Insert a markdown table", InsertText: "| ${1:Header 1} | ${2:Header 2} | ${3:Header 3} |\n| --- | --- | --- |\n| ${4:Cell} | ${5:Cell} | ${6:Cell} |\n", Category: "Advanced", SortOrder: "04a"},
	{Label: "/columns", Detail: "Columns", Description: "Multi-column layout", InsertText: "<div style=\"display: flex; gap: 1em;\">\n<div style=\"flex: 1;\">\n\n${1:Column 1}\n\n</div>\n<div style=\"flex: 1;\">\n\n${2:Column 2}\n\n</div>\n</div>\n", Category: "Advanced", SortOrder: "04b"},
	{Label: "/mermaid", Detail: "Mermaid diagram", Description: "Insert a Mermaid diagram", InsertText: "```mermaid\n${1|graph TD,sequenceDiagram,classDiagram,stateDiagram-v2,gantt,pie,flowchart|}\n  ${2:A --> B}\n```\n", Category: "Advanced", SortOrder: "04c"},
	{Label: "/frontmatter", Detail: "Frontmatter", Description: "YAML frontmatter block", InsertText: "---\ntitle: ${1:Title}\ndate: ${2:2024-01-01}\ntags: [${3:tag1, tag2}]\n---\n\n", Category: "Advanced", SortOrder: "04d"},
	{Label: "/footnote", Detail: "Footnote", Description: "Add a footnote reference", InsertText: "[^${1:ref}]: ${2:Footnote text}\n", Category: "Advanced", SortOrder: "04e"},
	{Label: "/toc", Detail: "Table of Contents", Description: "Auto-generated TOC marker", InsertText: "## Table of Contents\n\n[TOC]\n\n", Category: "Advanced", SortOrder: "04f"},

	// Database / structured data
	{Label: "/database", Detail: "Database view", Description: "Inline database table", InsertText: "---\ntype: database\nschema:\n  - name: ${1:Name}\n    type: text\n  - name: ${2:Status}\n    type: select\n    options: [todo, in-progress, done]\n  - name: ${3:Date}\n    type: date\n---\n\n| ${1:Name} | ${2:Status} | ${3:Date} |\n| --- | --- | --- |\n| ${4:Item 1} | todo | ${5:2024-01-01} |\n", Category: "Database", SortOrder: "05a"},
	{Label: "/kanban", Detail: "Kanban board", Description: "Kanban-style task board", InsertText: "## 📋 Kanban\n\n### Todo\n- [ ] ${1:Task 1}\n- [ ] ${2:Task 2}\n\n### In Progress\n- [ ] ${3:Task 3}\n\n### Done\n- [x] ${4:Task 4}\n", Category: "Database", SortOrder: "05b"},
	{Label: "/timeline", Detail: "Timeline", Description: "Timeline/gantt view", InsertText: "## 📅 Timeline\n\n| Date | Event |\n| --- | --- |\n| ${1:2024-01-01} | ${2:Event 1} |\n| ${3:2024-02-01} | ${4:Event 2} |\n", Category: "Database", SortOrder: "05c"},

	// Synced & Templates
	{Label: "/synced", Detail: "Synced block", Description: "Create a synced/transclusion block", InsertText: "<!-- sync:${1:block-id} -->\n${2:Synced content}\n<!-- /sync -->\n", Category: "Sync", SortOrder: "06a"},
	{Label: "/template", Detail: "Template", Description: "Insert from template", InsertText: "<!-- template:${1:template-name} -->\n${2:Template content}\n<!-- /template -->\n", Category: "Sync", SortOrder: "06b"},
	{Label: "/button", Detail: "Template button", Description: "Button that inserts a template", InsertText: "<!-- button: ${1:Button Label} | template: ${2:template-name} -->\n", Category: "Sync", SortOrder: "06c"},

	// Date & Time
	{Label: "/today", Detail: "Today's date", Description: "Insert today's date", InsertText: "${CURRENT_YEAR}-${CURRENT_MONTH}-${CURRENT_DATE}", Category: "Date", SortOrder: "07a"},
	{Label: "/now", Detail: "Current datetime", Description: "Insert current date and time", InsertText: "${CURRENT_YEAR}-${CURRENT_MONTH}-${CURRENT_DATE} ${CURRENT_HOUR}:${CURRENT_MINUTE}", Category: "Date", SortOrder: "07b"},
	{Label: "/due", Detail: "Due date", Description: "Insert a due date marker", InsertText: "📅 ${1:2024-01-01}", Category: "Date", SortOrder: "07c"},
	{Label: "/reminder", Detail: "Reminder", Description: "Set a reminder date", InsertText: "⏰ ${1:2024-01-01} ${2:09:00}", Category: "Date", SortOrder: "07d"},
	{Label: "/date", Detail: "Date", Description: "Insert current date", InsertText: "${CURRENT_YEAR}-${CURRENT_MONTH}-${CURRENT_DATE}", Category: "Date", SortOrder: "07e"},
	{Label: "/time", Detail: "Time", Description: "Insert current time", InsertText: "${CURRENT_HOUR}:${CURRENT_MINUTE}", Category: "Date", SortOrder: "07f"},

	// Inline
	{Label: "/mention", Detail: "Mention person", Description: "Mention a person", InsertText: "@${1:person}", Category: "Inline", SortOrder: "08a"},
	{Label: "/tag", Detail: "Add tag", Description: "Insert a tag", InsertText: "#${1:tag}", Category: "Inline", SortOrder: "08b"},
	{Label: "/highlight", Detail: "Highlight text", Description: "Highlight text with ==", InsertText: "==${1:highlighted text}==", Category: "Inline", SortOrder: "08c"},
	{Label: "/strikethrough", Detail: "Strikethrough", Description: "Strikethrough text", InsertText: "~~${1:text}~~", Category: "Inline", SortOrder: "08d"},
	{Label: "/inline-math", Detail: "Inline math", Description: "Inline LaTeX math", InsertText: "$$${1:formula}$$", Category: "Inline", SortOrder: "08e"},
}

// SlashCommandCompletions returns completions triggered by /commands.
// The query parameter is the text after the / that the user has typed so far.
func SlashCommandCompletions(items []protocol.CompletionItem, query string) []protocol.CompletionItem {
	snippetFormat := protocol.InsertTextFormatSnippet
	kind := protocol.CompletionItemKindSnippet

	for _, cmd := range SlashCommands {
		// Filter by query if provided
		if query != "" {
			label := cmd.Label[1:] // remove leading /
			if !containsIgnoreCase(label, query) && !containsIgnoreCase(cmd.Category, query) {
				continue
			}
		}

		detail := cmd.Category + ": " + cmd.Detail
		doc := protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "**" + cmd.Label + "** — " + cmd.Description + "\n\n```markdown\n" + stripSnippetVars(cmd.InsertText) + "\n```",
		}
		insertText := cmd.InsertText
		sortText := cmd.SortOrder

		items = append(items, protocol.CompletionItem{
			Label:            cmd.Label,
			Kind:             &kind,
			Detail:           &detail,
			Documentation:    &doc,
			InsertText:       &insertText,
			InsertTextFormat: &snippetFormat,
			SortText:         &sortText,
			FilterText:       &cmd.Label,
		})
	}

	return items
}

// containsIgnoreCase checks if haystack contains needle (case-insensitive).
func containsIgnoreCase(haystack, needle string) bool {
	h := make([]byte, len(haystack))
	n := make([]byte, len(needle))
	for i := range haystack {
		c := haystack[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		h[i] = c
	}
	for i := range needle {
		c := needle[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		n[i] = c
	}
	hs := string(h)
	ns := string(n)
	for i := 0; i <= len(hs)-len(ns); i++ {
		if hs[i:i+len(ns)] == ns {
			return true
		}
	}
	return false
}

// stripSnippetVars removes ${N:text} snippet placeholders for preview.
func stripSnippetVars(s string) string {
	result := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
			// Find the colon or pipe
			j := i + 2
			for j < len(s) && s[j] != ':' && s[j] != '|' && s[j] != '}' {
				j++
			}
			if j < len(s) && (s[j] == ':' || s[j] == '|') {
				// Extract default value
				k := j + 1
				depth := 1
				for k < len(s) && depth > 0 {
					if s[k] == '{' {
						depth++
					} else if s[k] == '}' {
						depth--
					}
					if depth > 0 {
						if s[k] != '|' {
							result = append(result, s[k])
						} else {
							// For choice snippets, take first option
							break
						}
					}
					k++
				}
				i = k + 1
				if i > len(s) {
					i = len(s)
				}
			} else if j < len(s) && s[j] == '}' {
				// No default, just skip
				i = j + 1
			} else {
				result = append(result, s[i])
				i++
			}
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}
