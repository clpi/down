package handler

import (
	"strings"

	"github.com/clpi/down/lsp/files"
	"github.com/clpi/down/lsp/handler/completion"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	Active          = protocol.UInteger(0)
	ActiveParameter = protocol.UInteger(0)
	TriggerCharacters = []string{
		"@", "/", "[", "(", "{", "|",
	}
	RetriggerCharacters = []string{
		"@", "/",
	}
	workDone = protocol.WorkDoneProgressOptions{
		WorkDoneProgress: &t,
	}
	SignatureOptions = protocol.SignatureHelpOptions{
		WorkDoneProgressOptions: workDone,
		TriggerCharacters:       TriggerCharacters,
		RetriggerCharacters:     RetriggerCharacters,
	}
	Registration = protocol.CompletionRegistrationOptions{
		TextDocumentRegistrationOptions: files.DocumentRegistration,
		CompletionOptions:               completion.Provider,
	}
)

func makeSignature(label, doc string, params ...protocol.ParameterInformation) protocol.SignatureInformation {
	return protocol.SignatureInformation{
		Label:           label,
		Documentation:   doc,
		ActiveParameter: &ActiveParameter,
		Parameters:      params,
	}
}

func makeParam(label, doc string) protocol.ParameterInformation {
	return protocol.ParameterInformation{
		Label:         label,
		Documentation: doc,
	}
}

func (s *State) SignatureHelp(c *glsp.Context, p *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
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
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	var sigs []protocol.SignatureInformation

	// Detect context: slash command, wiki link, markdown link, frontmatter, table
	if idx := strings.LastIndex(prefix, "/"); idx >= 0 && (idx == 0 || prefix[idx-1] == ' ' || prefix[idx-1] == '\t') {
		query := prefix[idx+1:]
		_ = query
		sigs = slashSignatures(sigs)
	} else if idx := strings.LastIndex(prefix, "[["); idx >= 0 && !strings.Contains(prefix[idx:], "]]") {
		sigs = append(sigs, makeSignature(
			"[[target|display text]]",
			"Wiki link: `target` is a page name, `|display text` is optional alias.\n\nUse to link to other workspace pages. Links are tracked in the knowledge graph.",
			makeParam("target", "Page name or entity to link to"),
			makeParam("|display text", "Optional display text shown instead of target name"),
		))
	} else if idx := strings.LastIndex(prefix, "["); idx >= 0 {
		sigs = append(sigs, makeSignature(
			"[text](url#anchor)",
			"Markdown link with optional anchor.\n\n- `text` is the displayed link text\n- `url` is the target (file path, URL, or heading anchor)\n- `#anchor` optionally links to a specific heading",
			makeParam("text", "The clickable link text"),
			makeParam("url", "Target URL, file path, or `#heading-slug`"),
		))
	} else if strings.HasPrefix(strings.TrimSpace(line), "|") {
		sigs = append(sigs, makeSignature(
			"| col1 | col2 | col3 |",
			"Markdown table row.\n\nUse | to separate columns. The header row sets column count. Alignment syntax:\n- `:---` left\n- `:---:` center\n- `---:` right",
			makeParam("| col1 |", "First column"),
			makeParam("| col2 |", "Additional columns separated by |"),
		))
	}

	if len(sigs) == 0 {
		return nil, nil
	}

	return &protocol.SignatureHelp{
		ActiveSignature: &Active,
		ActiveParameter: &ActiveParameter,
		Signatures:      sigs,
	}, nil
}

func slashSignatures(sigs []protocol.SignatureInformation) []protocol.SignatureInformation {
	return append(sigs,
		makeSignature(
			"/heading [level]",
			"Insert a heading. Level 1-6.",
			makeParam("level", "Heading level: 1-6 (default 2)"),
		),
		makeSignature(
			"/table [rows] [cols]",
			"Insert a markdown table with header row.",
			makeParam("rows", "Number of data rows"),
			makeParam("cols", "Number of columns"),
		),
		makeSignature(
			"/code [language]",
			"Insert a fenced code block.",
			makeParam("language", "Programming language for syntax highlighting"),
		),
		makeSignature(
			"/todo [text]",
			"Insert a task checkbox.",
			makeParam("text", "Task description"),
		),
		makeSignature(
			"/callout [type]",
			"Insert a callout block. Types: info, warning, error, tip, note.",
			makeParam("type", "Callout type: info, warning, error, tip, note"),
		),
		makeSignature(
			"/template [name]",
			"Insert a document template.",
			makeParam("name", "Template name from .down/templates/"),
		),
		makeSignature(
			"/database [name]",
			"Create a database (markdown table with schema).",
			makeParam("name", "Database name"),
		),
	)
}
