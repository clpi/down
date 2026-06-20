package handler

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/clpi/down/lsp/files"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var reLensTask = regexp.MustCompile(`^\s*[-*+]\s+\[([ xX])\]\s+(.*)`)
var reCodeBlock = regexp.MustCompile("^```")

var (
	LensProvider = protocol.CodeLensOptions{
		ResolveProvider: &trueVal,
		WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
			WorkDoneProgress: &trueVal,
		},
	}
	LensRegistration = protocol.CodeLensRegistrationOptions{
		TextDocumentRegistrationOptions: protocol.TextDocumentRegistrationOptions{
			DocumentSelector: &files.Filetypes,
		},
		CodeLensOptions: LensProvider,
	}
)

func (s *State) CodeLens(_ *glsp.Context, p *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return s.workspaceLenses(), nil
	}

	var lens []protocol.CodeLens

	lines := strings.Split(text, "\n")
	openCount := 0
	doneCount := 0
	codeBlocks := 0
	headings := 0

	for i, line := range lines {
		lineNum := i

		// Task codelens
		if m := reLensTask.FindStringSubmatch(line); m != nil {
			completed := m[1] != " "
			if completed {
				doneCount++
			} else {
				openCount++
			}
			title := "Toggle: " + m[2]
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			lens = append(lens, protocol.CodeLens{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
				},
				Command: &protocol.Command{
					Command:   "down.task.toggle",
					Title:     title,
					Arguments: []any{uri, lineNum},
				},
			})
		}

		// Code block run codelens
		if reCodeBlock.MatchString(strings.TrimSpace(line)) {
			if codeBlocks%2 == 0 && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				lang := extractLang(line)
				title := "Run"
				if lang != "" {
					title = "▶ Run " + lang
				}
				lens = append(lens, protocol.CodeLens{
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
						End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
					},
					Command: &protocol.Command{
						Command:   "down.code.run",
						Title:     title,
						Arguments: []any{uri, nextLine},
					},
				})
			}
			codeBlocks++
		}

		// Heading count
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			headings++
		}
	}

	// Summary codelenses at the top
	if openCount+doneCount > 0 {
		progress := 0
		if openCount+doneCount > 0 {
			progress = (doneCount * 100) / (openCount + doneCount)
		}
		lens = append(lens, protocol.CodeLens{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Command: &protocol.Command{
				Command: "down.task.list",
				Title:   fmt.Sprintf("Tasks: %d/%d done (%d%%)", doneCount, openCount+doneCount, progress),
			},
		})
	}

	if headings > 0 {
		lens = append(lens, protocol.CodeLens{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Command: &protocol.Command{
				Command: "down.toc.generate",
				Title:   fmt.Sprintf("Outline: %d headings", headings),
			},
		})
	}

	if codeBlocks > 0 {
		lens = append(lens, protocol.CodeLens{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Command: &protocol.Command{
				Command: "down.code.run",
				Title:   fmt.Sprintf("▶ Run blocks (%d code blocks)", codeBlocks/2),
			},
		})
	}

	// Knowledge graph codelens
	if s.Graph != nil {
		entities := s.Graph.EntitiesByDocument(uri)
		if len(entities) > 0 {
			lens = append(lens, protocol.CodeLens{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Command: &protocol.Command{
					Command: "down.knowledge.summary",
					Title:   fmt.Sprintf("Knowledge: %d entities", len(entities)),
				},
			})
		}
	}

	// Workspace lenses at the bottom
	lens = append(lens, s.workspaceLenses()...)

	return lens, nil
}

func (s *State) workspaceLenses() []protocol.CodeLens {
	return []protocol.CodeLens{
		{
			Command: &protocol.Command{
				Command: "down.sync",
				Title:   "🔄 Sync workspace",
			},
		},
		{
			Command: &protocol.Command{
				Command: "down.workspace.open",
				Title:   "📂 Open workspace",
			},
		},
		{
			Command: &protocol.Command{
				Command: "down.template.index",
				Title:   "📋 List templates",
			},
		},
	}
}

func extractLang(line string) string {
	rest := strings.TrimPrefix(strings.TrimSpace(line), "```")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	parts := strings.Fields(rest)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (s *State) LensResolve(_ *glsp.Context, p *protocol.CodeLens) (*protocol.CodeLens, error) {
	return p, nil
}
