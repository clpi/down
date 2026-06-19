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
	InsertText string          `json:"insertText"`
	Range      protocol.Range  `json:"range"`
	Command    *protocol.Command `json:"command,omitempty"`
	FilterText string          `json:"filterText,omitempty"`
}

// InlineCompletionParams mirrors the LSP 3.18 InlineCompletionParams.
type InlineCompletionParams struct {
	TextDocument protocol.TextDocumentIdentifier `json:"textDocument"`
	Position     protocol.Position               `json:"position"`
	Context      InlineCompletionContext         `json:"context"`
}

// InlineCompletionContext provides context for inline completion.
type InlineCompletionContext struct {
	TriggerKind   int    `json:"triggerKind"` // 1=Automatic, 2=Explicit
	SelectedText  string `json:"selectedCompletionInfo,omitempty"`
}

// InlineComplete provides AI-powered ghost text completions.
// This generates multi-line continuation suggestions based on document context.
// Available via the "down.inline.complete" command.
func (s *State) InlineComplete(_ *glsp.Context, p *protocol.ExecuteCommandParams) (any, error) {
	if s.AI == nil {
		return nil, nil
	}

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

// generateInlineCompletions uses the AI engine to produce ghost text suggestions.
func (s *State) generateInlineCompletions(ctx context.Context, docURI, preceding, currentLine, following string) []string {
	if s.AI == nil {
		return nil
	}

	completions, err := s.AI.InlineComplete(ctx, docURI, preceding, currentLine, following)
	if err != nil {
		return nil
	}
	return completions
}

// ComputeInlineCompletion provides inline ghost-text for a specific position.
// This is the main entry point for editors supporting custom inline completion.
func (s *State) ComputeInlineCompletion(uri string, line int, character int) []InlineCompletionItem {
	if s.AI == nil {
		return nil
	}

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
