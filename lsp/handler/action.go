package handler

import (
	"fmt"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type (
	any = interface{}
	Env = map[string]string
)

var (
	t              bool               = true
	f              bool               = false
	v              protocol.Integer   = 0
	src                                  = protocol.CodeActionKindSource
	qf                                   = protocol.CodeActionKindQuickFix
	ref                                  = protocol.CodeActionKindRefactor
	ActionProvider protocol.CodeActionOptions = protocol.CodeActionOptions{
		CodeActionKinds: []protocol.CodeActionKind{
			protocol.CodeActionKindSource,
			protocol.CodeActionKindQuickFix,
			protocol.CodeActionKindRefactor,
			protocol.CodeActionKindRefactorExtract,
			protocol.CodeActionKindRefactorInline,
			protocol.CodeActionKindRefactorRewrite,
			protocol.CodeActionKindSourceOrganizeImports,
		},
		WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
			WorkDoneProgress: &t,
		},
		ResolveProvider: &t,
	}
)

var (
	aiActionKind = protocol.CodeActionKindSource
	quickFixKind = protocol.CodeActionKindQuickFix
	refactorKind = protocol.CodeActionKindRefactor
)

func makeCommandAction(command, title string, kind protocol.CodeActionKind, args ...any) protocol.CodeAction {
	return protocol.CodeAction{
		Command: &protocol.Command{
			Command:   command,
			Title:     title,
			Arguments: args,
		},
		Kind:  &kind,
		Title: title,
	}
}

func makeEditAction(title string, kind protocol.CodeActionKind, edit protocol.WorkspaceEdit) protocol.CodeAction {
	return protocol.CodeAction{
		Kind:  &kind,
		Title: title,
		Edit:  &edit,
	}
}

func (s *State) CodeAction(_ *glsp.Context, p *protocol.CodeActionParams) (any, error) {
	actions := []protocol.CodeAction{}

	uri := string(p.TextDocument.URI)
	text, hasDoc := s.Documents[uri]

	hasSelection := p.Range.Start.Line != p.Range.End.Line ||
		p.Range.Start.Character != p.Range.End.Character

	// Always-available actions
	actions = append(actions,
		makeCommandAction("down.link.create.cursor", "Create wiki link at cursor", src),
		makeCommandAction("down.toc.generate", "Generate Table of Contents", src),
		makeCommandAction("down.template.new", "Create template from selection", src),
	)

	// Selection-based actions
	if hasSelection && hasDoc {
		lines := strings.Split(text, "\n")
		selText := getSelectedText(lines, p.Range)

		actions = append(actions,
			makeCommandAction("down.ai.query", "AI: Ask about selection", src, selText),
			makeCommandAction("down.ai.expand", "AI: Expand", src, selText),
			makeCommandAction("down.ai.summarize", "AI: Summarize", src, selText),
			makeCommandAction("down.ai.explain", "AI: Explain", src, selText),
		)

		// Extract selection to new note
		title := extractTitle(selText)
		actions = append(actions, makeCommandAction(
			"down.snippet.new", "Extract to new note", ref,
			title, selText,
		))

		// Wrap in callout
		actions = append(actions, makeCommandAction(
			"down.ai.transform", "Wrap in callout (NOTE)", ref,
			"callout-note", selText,
		))
		actions = append(actions, makeCommandAction(
			"down.ai.transform", "Wrap in callout (WARNING)", ref,
			"callout-warning", selText,
		))

		// Format as code block
		actions = append(actions, makeCommandAction(
			"down.ai.transform", "Format as code block", ref,
			"code-block", selText,
		))
	}

	// Line-based actions
	if hasDoc {
		lines := strings.Split(text, "\n")
		line := int(p.Range.Start.Line)
		if line < len(lines) {
			currentLine := lines[line]

			// Toggle task if on a task line
			if strings.Contains(currentLine, "- [ ]") || strings.Contains(currentLine, "- [x]") || strings.Contains(currentLine, "- [X]") {
				actions = append(actions, makeCommandAction(
					"down.task.toggle", "Toggle task status", qf,
					uri, line,
				))
			}

			// Add to memory
			if strings.TrimSpace(currentLine) != "" {
				actions = append(actions, makeCommandAction(
					"down.memory.new", "Save to memory", src,
					extractTitle(currentLine), currentLine,
				))
			}
		}
	}

	// Knowledge graph actions
	if s.Graph != nil && hasDoc {
		entities := s.Graph.EntitiesByDocument(uri)
		todoCount := 0
		for _, ent := range entities {
			if ent.Kind == "action" && ent.Properties["status"] == "todo" {
				todoCount++
			}
		}
		if todoCount > 0 {
			actions = append(actions, makeCommandAction(
				"down.ai.suggest", "AI: Suggest next steps ("+fmt.Sprintf("%d", todoCount)+" tasks)", src,
			))
		}

		actions = append(actions, makeCommandAction(
			"down.knowledge.search", "Search knowledge graph", src, uri,
		))
		actions = append(actions, makeCommandAction(
			"down.knowledge.related", "Show related entities", src, uri,
		))
	}

	// Workspace actions
	actions = append(actions,
		makeCommandAction("down.sync", "Sync workspace", src),
		makeCommandAction("down.template.index", "List templates", src),
		makeCommandAction("down.workspace.open", "Open workspace", src),
	)

	return actions, nil
}

func (s *State) ActionResolve(_ *glsp.Context, p *protocol.CodeAction) (*protocol.CodeAction, error) {
	return p, nil
}

func getSelectedText(lines []string, rng protocol.Range) string {
	if len(lines) == 0 {
		return ""
	}
	startLine := int(rng.Start.Line)
	endLine := int(rng.End.Line)
	if startLine >= len(lines) {
		startLine = len(lines) - 1
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	var parts []string
	for i := startLine; i <= endLine; i++ {
		line := lines[i]
		if i == startLine && i == endLine {
			col := int(rng.Start.Character)
			endCol := int(rng.End.Character)
			if col < len(line) && endCol <= len(line) {
				parts = append(parts, line[col:endCol])
			}
		} else if i == startLine {
			col := int(rng.Start.Character)
			if col < len(line) {
				parts = append(parts, line[col:])
			}
		} else if i == endLine {
			endCol := int(rng.End.Character)
			if endCol <= len(line) {
				parts = append(parts, line[:endCol])
			}
		} else {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

func extractTitle(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "\n"); idx > 0 {
		text = text[:idx]
	}
	if len(text) > 60 {
		text = text[:57] + "..."
	}
	return text
}
