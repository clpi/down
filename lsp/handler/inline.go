package handler

import (
	"context"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// InlineCompletionItem represents a ghost-text inline completion suggestion.
// This is exposed via the down.inline.complete command since protocol 3.16 doesn't
// have native inline completion support. Clients can use this via custom requests.
type InlineCompletionItem struct {
	InsertText string            `json:"insertText"`
	Range      protocol.Range    `json:"range"`
	Command    *protocol.Command `json:"command,omitempty"`
	FilterText string            `json:"filterText,omitempty"`
}

// InlineCompletionParams mirrors the LSP 3.18 InlineCompletionParams.
type InlineCompletionParams struct {
	TextDocument protocol.TextDocumentIdentifier `json:"textDocument"`
	Position     protocol.Position               `json:"position"`
	Context      InlineCompletionContext         `json:"context"`
}

// InlineCompletionContext provides context for inline completion.
type InlineCompletionContext struct {
	TriggerKind  int    `json:"triggerKind"` // 1=Automatic, 2=Explicit
	SelectedText string `json:"selectedCompletionInfo,omitempty"`
}

// InlineComplete provides AI-powered ghost text completions.
// This generates multi-line continuation suggestions based on document context.
// Available via the "down.inline.complete" command.
func (s *State) InlineComplete(_ *glsp.Context, p *protocol.ExecuteCommandParams) (any, error) {
	args := p.Arguments
	if len(args) < 2 {
		return nil, nil
	}

	uri, _ := args[0].(string)
	lineNum := 0
	if v, ok := args[1].(float64); ok {
		lineNum = int(v)
	}

	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	if lineNum >= len(lines) {
		return nil, nil
	}

	currentLine := lines[lineNum]

	// Don't trigger on empty lines or very short prefixes
	trimmed := strings.TrimSpace(currentLine)
	if len(trimmed) < 2 {
		return nil, nil
	}

	// Build context: preceding 30 lines
	start := lineNum - 30
	if start < 0 {
		start = 0
	}
	precedingText := strings.Join(lines[start:lineNum], "\n")

	// Get following context too (5 lines)
	end := lineNum + 5
	if end > len(lines) {
		end = len(lines)
	}
	followingText := ""
	if lineNum+1 < len(lines) {
		followingText = strings.Join(lines[lineNum+1:end], "\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*1e9)
	defer cancel()

	completions := s.generateInlineCompletions(ctx, uri, precedingText, currentLine, followingText)
	if len(completions) == 0 {
		return nil, nil
	}

	// Build inline completion items
	items := make([]InlineCompletionItem, 0, len(completions))
	for _, comp := range completions {
		items = append(items, InlineCompletionItem{
			InsertText: comp,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      protocol.UInteger(lineNum),
					Character: protocol.UInteger(len(currentLine)),
				},
				End: protocol.Position{
					Line:      protocol.UInteger(lineNum),
					Character: protocol.UInteger(len(currentLine)),
				},
			},
			FilterText: trimmed,
		})
	}

	return items, nil
}

// generateInlineCompletions uses the AI engine to produce ghost text suggestions,
// falling back to knowledge-graph continuations when AI is unavailable.
func (s *State) generateInlineCompletions(ctx context.Context, docURI, preceding, currentLine, following string) []string {
	if s.AI != nil {
		completions, err := s.AI.InlineComplete(ctx, docURI, preceding, currentLine, following)
		if err == nil && len(completions) > 0 {
			return completions
		}
	}
	return s.knowledgeInlineCompletions(docURI, preceding, currentLine)
}

// knowledgeInlineCompletions suggests continuations from the embedded knowledge graph.
func (s *State) knowledgeInlineCompletions(docURI, preceding, currentLine string) []string {
	if s.Graph == nil {
		return nil
	}

	query := strings.TrimSpace(currentLine)
	if query == "" {
		// Use last non-empty preceding line as query seed
		for i := len(strings.Split(preceding, "\n")) - 1; i >= 0; i-- {
			line := strings.TrimSpace(strings.Split(preceding, "\n")[i])
			if line != "" {
				query = line
				break
			}
		}
	}
	if len(query) < 3 {
		return nil
	}

	// Extract last word as entity search term
	words := strings.Fields(query)
	searchTerm := query
	if len(words) > 0 {
		searchTerm = words[len(words)-1]
	}
	if len(searchTerm) < 2 {
		return nil
	}

	results := s.Graph.Search(searchTerm)
	if len(results) == 0 {
		return nil
	}

	completions := make([]string, 0, 3)
	seen := make(map[string]bool)

	// Prefer entities related to the current document
	docEntities := s.Graph.EntitiesByDocument(docURI)
	docRelated := make(map[string]bool)
	for _, ent := range docEntities {
		for _, rel := range s.Graph.RelationsFrom(ent.ID) {
			if target, ok := s.Graph.Entities[rel.To]; ok {
				docRelated[target.Name] = true
			}
		}
	}

	for _, ent := range results {
		if ent.Name == "" || seen[ent.Name] {
			continue
		}
		seen[ent.Name] = true

		suggestion := ""
		switch ent.Kind {
		case "tag":
			suggestion = " #" + ent.Name
		case "person", "mention":
			suggestion = " @" + ent.Name
		case "link", "document":
			suggestion = " [[" + ent.Name + "]]"
		default:
			if docRelated[ent.Name] {
				suggestion = " — see [[" + ent.Name + "]]"
			} else if ent.Properties["summary"] != "" {
				suggestion = " (" + truncate(ent.Properties["summary"], 60) + ")"
			} else {
				suggestion = " [[" + ent.Name + "]]"
			}
		}

		if suggestion != "" && !strings.Contains(currentLine, ent.Name) {
			completions = append(completions, suggestion)
		}
		if len(completions) >= 3 {
			break
		}
	}

	return completions
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// ComputeInlineCompletion provides inline ghost-text for a specific position.
// This is the main entry point for editors supporting custom inline completion.
func (s *State) ComputeInlineCompletion(uri string, line int, character int) []InlineCompletionItem {
	text, ok := s.Documents[uri]
	if !ok {
		return nil
	}

	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return nil
	}

	currentLine := lines[line]
	col := character
	if col > len(currentLine) {
		col = len(currentLine)
	}
	linePrefix := currentLine[:col]

	// Don't trigger on very short prefixes or inside code blocks
	if len(strings.TrimSpace(linePrefix)) < 3 {
		return nil
	}

	// Check if inside a code block
	inCode := false
	for i := 0; i < line; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inCode = !inCode
		}
	}
	if inCode {
		return nil
	}

	start := line - 30
	if start < 0 {
		start = 0
	}
	precedingText := strings.Join(lines[start:line], "\n")

	end := line + 5
	if end > len(lines) {
		end = len(lines)
	}
	followingText := ""
	if line+1 < len(lines) {
		followingText = strings.Join(lines[line+1:end], "\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*1e9)
	defer cancel()

	completions := s.generateInlineCompletions(ctx, uri, precedingText, linePrefix, followingText)
	if len(completions) == 0 {
		return nil
	}

	items := make([]InlineCompletionItem, 0, len(completions))
	for _, comp := range completions {
		items = append(items, InlineCompletionItem{
			InsertText: comp,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      protocol.UInteger(line),
					Character: protocol.UInteger(col),
				},
				End: protocol.Position{
					Line:      protocol.UInteger(line),
					Character: protocol.UInteger(col),
				},
			},
			FilterText: linePrefix,
		})
	}
	return items
}
