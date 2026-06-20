package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

// PredefinedDocTemplate represents a built-in document template.
type PredefinedDocTemplate struct {
	Name, Description, Content string
}

var DefaultDocTemplates = []PredefinedDocTemplate{
	{"Meeting Notes", "Standard meeting notes with agenda and action items", "# Meeting: {{title}}\n\n**Date:** {{date}}\n**Attendees:** \n\n## Agenda\n\n- \n\n## Notes\n\n## Action Items\n\n- [ ] \n"},
	{"Daily Journal", "Daily journal entry with reflection prompts", "# {{date}}\n\n## Morning\n\n## Work Log\n\n- \n\n## Evening Reflection\n\n## Gratitude\n\n- \n"},
	{"Project Overview", "Project overview with goals, timeline, and status", "# Project: {{title}}\n\n**Status:** in-progress\n**Start:** {{date}}\n**Target:** \n\n## Goals\n\n- \n\n## Timeline\n\n| Phase | Status | Date |\n|-------|--------|------|\n| Planning | done | |\n| Execution | in-progress | |\n| Review | pending | |\n\n## Notes\n"},
	{"Research Note", "Research note with methodology and findings", "# Research: {{title}}\n\n**Date:** {{date}}\n**Source:** \n\n## Key Findings\n\n- \n\n## Methodology\n\n## Raw Notes\n\n## References\n\n- \n"},
	{"Code Snippet", "Code snippet documentation with usage notes", "# `{{title}}`\n\n**Language:** \n**Source:** \n\n## Description\n\n## Usage\n\n```\n\n```\n\n## Notes\n"},
}

func DocTemplateItems(i []protocol.CompletionItem) []protocol.CompletionItem {
	items := append([]protocol.CompletionItem{}, i...)
	kind := protocol.CompletionItemKindSnippet
	for _, t := range DefaultDocTemplates {
		insert := t.Content
		detail := t.Description
		items = append(items, protocol.CompletionItem{
			Label:            t.Name,
			InsertText:       &insert,
			InsertTextFormat: &SnippetFormat,
			Kind:             &kind,
			Detail:           &detail,
			Documentation:    Documentation(t.Description),
			CommitCharacters: CommitCharacters,
		})
	}
	return items
}
