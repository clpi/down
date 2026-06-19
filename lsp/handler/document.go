package handler

import (
	"regexp"
	"strings"

	files "github.com/clpi/down/lsp/files"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	file       = "file"
	partialTok = "1"
)

type (
	DocProvider struct {
		Registration       protocol.TextDocumentRegistrationOptions
		Link               protocol.DocumentLinkOptions
		Highlight          protocol.DocumentHighlightOptions
		Implementation     protocol.ImplementationOptions
		References         protocol.ReferenceParams
		Declaration        protocol.DeclarationOptions
		Definition         protocol.DefinitionOptions
		TypeDefinition     protocol.TypeDefinitionOptions
		LinkedEditingRange protocol.LinkedEditingRangeOptions
		Moniker            protocol.MonikerOptions
		Symbol             protocol.DocumentSymbolOptions
		Color              protocol.DocumentColorOptions
		Format             protocol.DocumentFormattingOptions
		RangeFormat        protocol.DocumentRangeFormattingOptions
		OnType             protocol.DocumentOnTypeFormattingOptions
		Sync               protocol.TextDocumentSyncOptions
		Hover              protocol.HoverOptions
	}
)

var DocumentProvider = DocProvider{
	Registration: protocol.TextDocumentRegistrationOptions{
		DocumentSelector: &files.Filetypes,
	},
	Sync: protocol.TextDocumentSyncOptions{
		OpenClose:         &trueVal,
		WillSave:          &trueVal,
		WillSaveWaitUntil: &trueVal,
		Save: &protocol.SaveOptions{
			IncludeText: &trueVal,
		},
	},
	Highlight: protocol.DocumentHighlightOptions{
		WorkDoneProgressOptions: workDone,
	},
	Implementation: protocol.ImplementationOptions{
		WorkDoneProgressOptions: workDone,
	},
	References: protocol.ReferenceParams{
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{
			WorkDoneToken: &protocol.ProgressToken{
				Value: partialTok,
			},
		},
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "",
			},
			Position: protocol.Position{
				Line:      0,
				Character: 0,
			},
		},
		PartialResultParams: protocol.PartialResultParams{
			PartialResultToken: &protocol.ProgressToken{"1"},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	},
	Declaration: protocol.DeclarationOptions{
		WorkDoneProgressOptions: workDone,
	},
	Definition: protocol.DefinitionOptions{
		WorkDoneProgressOptions: workDone,
	},
	TypeDefinition: protocol.TypeDefinitionOptions{
		WorkDoneProgressOptions: workDone,
	},
	LinkedEditingRange: protocol.LinkedEditingRangeOptions{
		WorkDoneProgressOptions: workDone,
	},
	Moniker: protocol.MonikerOptions{
		WorkDoneProgressOptions: workDone,
	},
	Symbol: protocol.DocumentSymbolOptions{
		WorkDoneProgressOptions: workDone,
	},
	Hover: protocol.HoverOptions{
		WorkDoneProgressOptions: workDone,
	},
	Link: protocol.DocumentLinkOptions{
		ResolveProvider:         &trueVal,
		WorkDoneProgressOptions: workDone,
	},
	Color: protocol.DocumentColorOptions{
		WorkDoneProgressOptions: workDone,
	},
	Format: protocol.DocumentFormattingOptions{
		WorkDoneProgressOptions: workDone,
	},
	RangeFormat: protocol.DocumentRangeFormattingOptions{
		WorkDoneProgressOptions: workDone,
	},
	OnType: protocol.DocumentOnTypeFormattingOptions{
		FirstTriggerCharacter: " ",
		MoreTriggerCharacter:  []string{" ", "\n"},
	},
}

func (s *State) DidSave(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	if params.Text != nil {
		s.Documents[uri] = *params.Text
		knowledge.ExtractFromDocument(s.Graph, uri, *params.Text)
		s.Graph.Save()
		s.publishDiagnostics(ctx, uri, *params.Text)
	}
	return nil
}

func (s *State) DidClose(ctx *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	delete(s.Documents, uri)
	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentUri(uri),
		Diagnostics: []protocol.Diagnostic{},
	})
	return nil
}

func (s *State) WillSaveWaitUntil(context *glsp.Context, params *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, nil
}

func (s *State) WillSave(context *glsp.Context, params *protocol.WillSaveTextDocumentParams) error {
	return nil
}

func (s *State) DidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	text := params.TextDocument.Text
	s.Documents[uri] = text
	knowledge.ExtractFromDocument(s.Graph, uri, text)
	s.publishDiagnostics(ctx, uri, text)
	return nil
}

func (s *State) DidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			s.Documents[uri] = c.Text
			knowledge.ExtractFromDocument(s.Graph, uri, c.Text)
		}
	}
	return nil
}

func (s *State) Rename(_ *glsp.Context, p *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	line := int(p.Position.Line)
	if line >= len(lines) {
		return nil, nil
	}

	word := wordAtPosition(lines[line], int(p.Position.Character))
	if word == "" {
		return nil, nil
	}

	newName := p.NewName

	// Find all documents containing this word and build edits
	changes := map[protocol.DocumentUri][]protocol.TextEdit{}
	for docURI, docText := range s.Documents {
		docLines := strings.Split(docText, "\n")
		for lineIdx, l := range docLines {
			col := 0
			lower := strings.ToLower(l)
			target := strings.ToLower(word)
			for {
				idx := strings.Index(lower[col:], target)
				if idx < 0 {
					break
				}
				pos := col + idx
				// Check word boundaries
				if pos > 0 && isWordChar(l[pos-1]) {
					col = pos + 1
					continue
				}
				end := pos + len(word)
				if end < len(l) && isWordChar(l[end]) {
					col = pos + 1
					continue
				}
				changes[protocol.DocumentUri(docURI)] = append(
					changes[protocol.DocumentUri(docURI)],
					protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: protocol.UInteger(lineIdx), Character: protocol.UInteger(pos)},
							End:   protocol.Position{Line: protocol.UInteger(lineIdx), Character: protocol.UInteger(end)},
						},
						NewText: newName,
					},
				)
				col = end
			}
		}
	}

	if len(changes) == 0 {
		return nil, nil
	}

	return &protocol.WorkspaceEdit{Changes: changes}, nil
}

func (s *State) PrepareRename(_ *glsp.Context, p *protocol.PrepareRenameParams) (any, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	line := int(p.Position.Line)
	if line >= len(lines) {
		return nil, nil
	}

	word := wordAtPosition(lines[line], int(p.Position.Character))
	if word == "" {
		return nil, nil
	}

	// Find the word start position
	col := int(p.Position.Character)
	start := col
	for start > 0 && isWordChar(lines[line][start-1]) {
		start--
	}

	return protocol.Range{
		Start: protocol.Position{Line: p.Position.Line, Character: protocol.UInteger(start)},
		End:   protocol.Position{Line: p.Position.Line, Character: protocol.UInteger(start + len(word))},
	}, nil
}

func wordAtPosition(line string, col int) string {
	if col >= len(line) {
		return ""
	}
	start := col
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	end := col
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	if start == end {
		return ""
	}
	return line[start:end]
}

// Moniker implements textDocument/moniker.
// Monikers provide stable, cross-document identifiers for entities extracted from
// the knowledge graph. This enables cross-repository navigation and global symbol identification.
// Moniker schemes:
//   - "down:entity" — knowledge graph entities (tags, people, concepts, projects)
//   - "down:doc" — document-level identifiers
//   - "down:heading" — heading-level anchors
func (s *State) Moniker(_ *glsp.Context, p *protocol.MonikerParams) ([]protocol.Moniker, error) {
	if s.Graph == nil {
		return nil, nil
	}

	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil, nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col >= len(line) {
		return nil, nil
	}

	var monikers []protocol.Moniker

	// Check if cursor is on a wiki link [[target]]
	for _, m := range reLinkWiki.FindAllStringSubmatchIndex(line, -1) {
		innerStart, innerEnd := m[2], m[3]
		if col >= m[0] && col <= m[1] {
			target := line[innerStart:innerEnd]
			parts := strings.SplitN(target, "|", 2)
			linkTarget := strings.TrimSpace(parts[0])
			monikers = append(monikers, protocol.Moniker{
				Scheme:     "down",
				Identifier: "entity:" + strings.ToLower(linkTarget),
				Unique:     protocol.UniquenessLevelGlobal,
				Kind:       &monikerKindImport,
			})
			return monikers, nil
		}
	}

	// Check if cursor is on a #tag
	for _, m := range reSemanticTag.FindAllStringSubmatchIndex(line, -1) {
		tagStart := m[0]
		for tagStart < m[1] && line[tagStart] != '#' {
			tagStart++
		}
		if col >= tagStart && col <= m[1] {
			tag := line[tagStart+1 : m[1]]
			monikers = append(monikers, protocol.Moniker{
				Scheme:     "down",
				Identifier: "tag:" + strings.ToLower(tag),
				Unique:     protocol.UniquenessLevelGlobal,
				Kind:       &monikerKindExport,
			})
			return monikers, nil
		}
	}

	// Check if cursor is on a @mention
	for _, m := range reSemanticMention.FindAllStringSubmatchIndex(line, -1) {
		mentionStart := m[0]
		for mentionStart < m[1] && line[mentionStart] != '@' {
			mentionStart++
		}
		if col >= mentionStart && col <= m[1] {
			mention := line[mentionStart+1 : m[1]]
			monikers = append(monikers, protocol.Moniker{
				Scheme:     "down",
				Identifier: "person:" + strings.ToLower(mention),
				Unique:     protocol.UniquenessLevelGlobal,
				Kind:       &monikerKindImport,
			})
			return monikers, nil
		}
	}

	// Check if cursor is on a heading
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		level := 0
		for _, ch := range trimmed {
			if ch == '#' {
				level++
			} else {
				break
			}
		}
		headingText := strings.TrimSpace(trimmed[level:])
		slug := monikerSlug(headingText)
		monikers = append(monikers, protocol.Moniker{
			Scheme:     "down",
			Identifier: "heading:" + slug,
			Unique:     protocol.UniquenessLevelDocument,
			Kind:       &monikerKindExport,
		})
		return monikers, nil
	}

	// Fall back: check if cursor is on a known entity word
	wordStart, wordEnd := col, col
	for wordStart > 0 && isWordChar(line[wordStart-1]) {
		wordStart--
	}
	for wordEnd < len(line) && isWordChar(line[wordEnd]) {
		wordEnd++
	}
	if wordStart >= wordEnd {
		return nil, nil
	}
	word := line[wordStart:wordEnd]

	results := s.Graph.Search(word)
	for _, ent := range results {
		if strings.EqualFold(ent.Name, word) {
			monikers = append(monikers, protocol.Moniker{
				Scheme:     "down",
				Identifier: string(ent.Kind) + ":" + strings.ToLower(ent.Name),
				Unique:     protocol.UniquenessLevelGlobal,
				Kind:       &monikerKindImport,
			})
			break
		}
	}

	if len(monikers) == 0 {
		return nil, nil
	}
	return monikers, nil
}

var (
	monikerKindExport = protocol.MonikerKindExport
	monikerKindImport = protocol.MonikerKindImport

	reSemanticTag     = reLinkWiki // reuse — will define proper ones below
	reSemanticMention = reLinkMd   // placeholder — override below
)

func init() {
	// Use the actual regex patterns for tag and mention detection in monikers
	reSemanticTag = regexp.MustCompile(`(?:^|\s)#([a-zA-Z][a-zA-Z0-9_/-]*)`)
	reSemanticMention = regexp.MustCompile(`(?:^|\s)@([a-zA-Z][a-zA-Z0-9_.-]*)`)
}

// monikerSlug creates a URL-safe slug from heading text.
func monikerSlug(text string) string {
	text = strings.ToLower(text)
	var result []byte
	for _, b := range []byte(text) {
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') {
			result = append(result, b)
		} else if b == ' ' || b == '-' || b == '_' {
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	// Trim trailing dashes
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return string(result)
}

func (s *State) References(_ *glsp.Context, p *protocol.ReferenceParams) ([]protocol.Location, error) {
	if s.Graph == nil {
		return nil, nil
	}

	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil, nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col >= len(line) {
		return nil, nil
	}

	wordStart, wordEnd := col, col
	for wordStart > 0 && isWordChar(line[wordStart-1]) {
		wordStart--
	}
	for wordEnd < len(line) && isWordChar(line[wordEnd]) {
		wordEnd++
	}
	if wordStart >= wordEnd {
		return nil, nil
	}
	word := line[wordStart:wordEnd]

	results := s.Graph.Search(word)
	var locations []protocol.Location
	for _, ent := range results {
		if strings.ToLower(ent.Name) != strings.ToLower(word) {
			continue
		}
		for _, src := range ent.Sources {
			if !p.Context.IncludeDeclaration && src.URI == uri && src.Line == lineIdx {
				continue
			}
			locations = append(locations, protocol.Location{
				URI: protocol.DocumentUri(src.URI),
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(src.Line), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(src.Line), Character: protocol.UInteger(len(ent.Name))},
				},
			})
		}
	}
	return locations, nil
}

func (s *State) Color(c *glsp.Context, p *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return []protocol.ColorInformation{}, nil
}

func (s *State) ColorPresentation(c *glsp.Context, p *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return []protocol.ColorPresentation{}, nil
}

func (s *State) Symbol(_ *glsp.Context, p *protocol.DocumentSymbolParams) (any, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")

	// Build a hierarchical document outline from headings, tasks, and entities.
	// This matches Notion's sidebar outline with nested sections.
	type headingNode struct {
		symbol   protocol.DocumentSymbol
		level    int
		children []protocol.DocumentSymbol
	}

	var rootSymbols []protocol.DocumentSymbol

	// Stack for building hierarchy
	type stackEntry struct {
		level    int
		symbols  *[]protocol.DocumentSymbol
	}
	stack := []stackEntry{{level: 0, symbols: &rootSymbols}}

	inCode := false
	inFrontmatter := false
	taskIdx := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track frontmatter
		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
				// Add frontmatter as a symbol
				detail := "metadata"
				rootSymbols = append(rootSymbols, protocol.DocumentSymbol{
					Name:   "Frontmatter",
					Detail: &detail,
					Kind:   protocol.SymbolKindPackage,
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 3},
					},
				})
			}
			continue
		}

		// Track code blocks
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}

		// Headings → hierarchical outline
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for _, ch := range trimmed {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			if level >= 1 && level <= 6 {
				headingText := strings.TrimSpace(trimmed[level:])
				if headingText == "" {
					continue
				}

				// Find the end of this section (next heading of same or higher level)
				endLine := len(lines) - 1
				for j := i + 1; j < len(lines); j++ {
					jt := strings.TrimSpace(lines[j])
					if strings.HasPrefix(jt, "#") {
						jLevel := 0
						for _, ch := range jt {
							if ch == '#' {
								jLevel++
							} else {
								break
							}
						}
						if jLevel <= level {
							endLine = j - 1
							break
						}
					}
				}
				// Trim trailing blanks
				for endLine > i && strings.TrimSpace(lines[endLine]) == "" {
					endLine--
				}

				detail := strings.Repeat("#", level)
				sym := protocol.DocumentSymbol{
					Name:   headingText,
					Detail: &detail,
					Kind:   symbolKindForHeadingLevel(level),
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
						End:   protocol.Position{Line: protocol.UInteger(endLine), Character: protocol.UInteger(len(lines[endLine]))},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
						End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
					},
				}

				// Pop stack until we find a parent with lower level
				for len(stack) > 1 && stack[len(stack)-1].level >= level {
					stack = stack[:len(stack)-1]
				}

				parent := stack[len(stack)-1].symbols
				*parent = append(*parent, sym)
				// Push this heading as new parent for children
				idx := len(*parent) - 1
				stack = append(stack, stackEntry{
					level:   level,
					symbols: &(*parent)[idx].Children,
				})
			}
		}

		// Tasks → nested under current heading
		if strings.Contains(trimmed, "- [ ]") || strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
			taskIdx++
			done := strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]")
			taskText := trimmed
			if idx := strings.Index(taskText, "] "); idx >= 0 {
				taskText = taskText[idx+2:]
			}

			kind := protocol.SymbolKindEvent
			detail := "todo"
			if done {
				detail = "done"
			}

			sym := protocol.DocumentSymbol{
				Name:   taskText,
				Detail: &detail,
				Kind:   kind,
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
				},
				SelectionRange: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
				},
			}
			if done {
				dep := true
				sym.Deprecated = &dep
			}

			parent := stack[len(stack)-1].symbols
			*parent = append(*parent, sym)
		}
	}

	if len(rootSymbols) == 0 {
		return nil, nil
	}
	return rootSymbols, nil
}

// symbolKindForHeadingLevel returns appropriate SymbolKind based on heading depth.
func symbolKindForHeadingLevel(level int) protocol.SymbolKind {
	switch level {
	case 1:
		return protocol.SymbolKindModule
	case 2:
		return protocol.SymbolKindClass
	case 3:
		return protocol.SymbolKindMethod
	case 4:
		return protocol.SymbolKindFunction
	case 5:
		return protocol.SymbolKindField
	default:
		return protocol.SymbolKindVariable
	}
}

func (s *State) Definition(_ *glsp.Context, p *protocol.DefinitionParams) (any, error) {
	if s.Graph == nil {
		return nil, nil
	}

	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil, nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col >= len(line) {
		return nil, nil
	}

	wordStart, wordEnd := col, col
	for wordStart > 0 && isWordChar(line[wordStart-1]) {
		wordStart--
	}
	for wordEnd < len(line) && isWordChar(line[wordEnd]) {
		wordEnd++
	}
	if wordStart >= wordEnd {
		return nil, nil
	}
	word := line[wordStart:wordEnd]

	results := s.Graph.Search(word)
	if len(results) == 0 {
		return nil, nil
	}

	var locations []protocol.Location
	for _, ent := range results {
		if strings.ToLower(ent.Name) != strings.ToLower(word) {
			continue
		}
		for _, src := range ent.Sources {
			if src.URI == uri && src.Line == lineIdx {
				continue
			}
			locations = append(locations, protocol.Location{
				URI: protocol.DocumentUri(src.URI),
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(src.Line), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(src.Line), Character: protocol.UInteger(len(ent.Name))},
				},
			})
		}
	}

	if len(locations) == 0 {
		return nil, nil
	}
	if len(locations) == 1 {
		return locations[0], nil
	}
	return locations, nil
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.'
}

func (s *State) publishDiagnostics(ctx *glsp.Context, uri string, text string) {
	if s.Graph == nil {
		return
	}

	kDiags := knowledge.DiagnoseDocument(s.Graph, uri, text)
	lspDiags := make([]protocol.Diagnostic, len(kDiags))

	sevMap := map[knowledge.DiagSeverity]protocol.DiagnosticSeverity{
		knowledge.SeverityError:   protocol.DiagnosticSeverityError,
		knowledge.SeverityWarning: protocol.DiagnosticSeverityWarning,
		knowledge.SeverityInfo:    protocol.DiagnosticSeverityInformation,
		knowledge.SeverityHint:    protocol.DiagnosticSeverityHint,
	}

	source := "down"
	for i, d := range kDiags {
		sev := sevMap[d.Severity]
		lspDiags[i] = protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(d.Line), Character: protocol.UInteger(d.ColStart)},
				End:   protocol.Position{Line: protocol.UInteger(d.Line), Character: protocol.UInteger(d.ColEnd)},
			},
			Severity: &sev,
			Source:   &source,
			Message:  d.Message,
		}
	}

	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentUri(uri),
		Diagnostics: lspDiags,
	})
}
