package handler

import (
	"context"
	"strings"

	"github.com/clpi/down/lsp/handler/completion/entries"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *State) Completion(
	c *glsp.Context,
	p *protocol.CompletionParams,
) (interface{}, error) {
	items := []protocol.CompletionItem{}

	// Detect slash command trigger
	slashQuery := s.detectSlashCommand(p)
	if slashQuery != nil {
		// Slash command mode: only return slash commands
		items = entries.SlashCommandCompletions(items, *slashQuery)
		return items, nil
	}

	// Detect tag command trigger
	tagQuery := s.detectTagCommand(p)
	if tagQuery != nil {
		items = entries.TagCompletions(items, s.Graph, *tagQuery)
		return items, nil
	}

	// Detect @mention trigger
	mentionQuery := s.detectMentionCommand(p)
	if mentionQuery != nil {
		items = entries.MentionCompletions(items, s.Graph, *mentionQuery)
		return items, nil
	}

	// Detect [[wiki link trigger
	wikiQuery := s.detectWikiLinkCommand(p)
	if wikiQuery != nil {
		items = entries.WikiLinkCompletions(items, s.Graph, *wikiQuery)
		return items, nil
	}

	items = entries.SnippetCompletions(items)
	items = entries.EmojiCompletions(items)
	items = entries.FileCompletions(items)
	items = entries.HtmlTagCompletions(items)
	items = entries.WorkspaceCompletions(items)
	items = s.knowledgeCompletions(items, p)
	items = s.aiCompletions(items, p)
	return items, nil
}

// detectSlashCommand checks if the user is typing a /command at the beginning of a line.
// Returns the query string after / if triggered, or nil if not a slash command context.
func (s *State) detectSlashCommand(p *protocol.CompletionParams) *string {
	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	// Slash commands trigger when / is at the start of a line (optionally preceded by whitespace)
	trimmed := strings.TrimLeft(prefix, " \t")
	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}

	// Extract the query after /
	query := trimmed[1:]

	// Don't trigger if it looks like a file path (contains another /)
	if strings.Contains(query, "/") {
		return nil
	}

	return &query
}

// detectTagCommand checks if the user is typing a #tag.
func (s *State) detectTagCommand(p *protocol.CompletionParams) *string {
	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	idx := strings.LastIndex(prefix, "#")
	if idx == -1 {
		return nil
	}

	// Must not be followed by space (headings are `# `)
	if idx+1 < len(prefix) && prefix[idx+1] == ' ' {
		return nil
	}

	// Make sure the `#` is preceded by whitespace or start of line
	if idx > 0 {
		charBefore := prefix[idx-1]
		if charBefore != ' ' && charBefore != '\t' && charBefore != '\n' && charBefore != '(' && charBefore != '[' {
			return nil
		}
	}

	query := prefix[idx+1:]
	// query should only contain valid characters
	for _, c := range query {
		if c == ' ' || c == '\t' || c == '\n' {
			return nil
		}
	}

	return &query
}


// detectMentionCommand checks if the user is typing an @mention.
func (s *State) detectMentionCommand(p *protocol.CompletionParams) *string {
	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	idx := strings.LastIndex(prefix, "@")
	if idx == -1 {
		return nil
	}

	if idx+1 < len(prefix) && prefix[idx+1] == ' ' {
		return nil
	}

	if idx > 0 {
		charBefore := prefix[idx-1]
		if charBefore != ' ' && charBefore != '\t' && charBefore != '\n' && charBefore != '(' && charBefore != '[' {
			return nil
		}
	}

	query := prefix[idx+1:]
	for _, c := range query {
		if c == ' ' || c == '\t' || c == '\n' {
			return nil
		}
	}

	return &query
}

// detectWikiLinkCommand checks if the user is typing inside an unclosed [[wiki link.
func (s *State) detectWikiLinkCommand(p *protocol.CompletionParams) *string {
	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	openIdx := strings.LastIndex(prefix, "[[")
	if openIdx == -1 {
		return nil
	}

	segment := prefix[openIdx+2:]
	if strings.Contains(segment, "]]") {
		return nil
	}

	for _, c := range segment {
		if c == ' ' || c == '\t' || c == '\n' {
			return nil
		}
	}

	query := segment
	if pipe := strings.Index(query, "|"); pipe >= 0 {
		query = query[:pipe]
	}

	return &query
}

func (s *State) knowledgeCompletions(items []protocol.CompletionItem, p *protocol.CompletionParams) []protocol.CompletionItem {
	if s.Graph == nil {
		return items
	}

	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return items
	}

	lines := strings.Split(doc, "\n")
	if int(p.Position.Line) >= len(lines) {
		return items
	}
	line := lines[p.Position.Line]
	col := int(p.Position.Character)
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	var query string
	if idx := strings.LastIndexAny(prefix, " \t@#["); idx >= 0 {
		query = prefix[idx+1:]
	} else {
		query = prefix
	}

	if len(query) < 2 {
		return items
	}

	results := s.Graph.Search(query)
	kindAI := protocol.CompletionItemKindReference
	for _, ent := range results {
		detail := string(ent.Kind)
		if len(ent.Sources) > 1 {
			detail += " (referenced in multiple docs)"
		}
		items = append(items, protocol.CompletionItem{
			Label:  ent.Name,
			Kind:   &kindAI,
			Detail: &detail,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: entityDoc(ent),
			},
		})
	}
	return items
}

func entityDoc(ent *knowledge.Entity) string {
	var sb strings.Builder
	sb.WriteString("**" + ent.Name + "** (`" + string(ent.Kind) + "`)\n\n")
	if len(ent.Properties) > 0 {
		for k, v := range ent.Properties {
			sb.WriteString("- " + k + ": " + v + "\n")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Mentions: " + intStr(ent.Mentions) + "\n")
	return sb.String()
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func (s *State) aiCompletions(items []protocol.CompletionItem, p *protocol.CompletionParams) []protocol.CompletionItem {
	if s.AI == nil {
		return items
	}

	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return items
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return items
	}

	currentLine := lines[lineIdx]
	col := int(p.Position.Character)
	if col > len(currentLine) {
		col = len(currentLine)
	}
	linePrefix := currentLine[:col]

	if len(strings.TrimSpace(linePrefix)) < 3 {
		return items
	}

	start := lineIdx - 20
	if start < 0 {
		start = 0
	}
	precedingText := strings.Join(lines[start:lineIdx], "\n")

	ctx, cancel := context.WithTimeout(context.Background(), 5*1e9)
	defer cancel()

	completions, err := s.AI.CompleteText(ctx, uri, precedingText, linePrefix)
	if err != nil {
		return items
	}

	kindAI := protocol.CompletionItemKindText
	for i, comp := range completions {
		label := comp
		if len(label) > 60 {
			label = label[:57] + "..."
		}
		detail := "AI suggestion"
		sortText := "zzz" + string(rune('0'+i))
		items = append(items, protocol.CompletionItem{
			Label:      label,
			Kind:       &kindAI,
			Detail:     &detail,
			InsertText: &comp,
			SortText:   &sortText,
		})
	}
	return items
}

func (s *State) ItemResolve(c *glsp.Context, p *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return p, nil
}
