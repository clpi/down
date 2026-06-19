package semantic

import (
	"github.com/clpi/down/lsp/files"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Token type indices — must match order in TokenTypes slice.
const (
	TokHeading    = iota // heading → "namespace"
	TokTag               // #tag → "macro"
	TokMention           // @mention → "variable"
	TokWikiLink          // [[link]] → "class"
	TokMdLink            // [text](url) → "function"
	TokTask              // - [ ] / - [x] → "event"
	TokCodeSpan          // `code` → "string"
	TokDate              // 2024-01-15 → "number"
	TokFrontmatter       // YAML frontmatter key → "property"
	TokBlockquote        // > quote → "comment"
	TokBold              // **bold** → "keyword"
	TokItalic            // *italic* → "modifier"
	TokEntity            // known knowledge graph entity → "type"
	TokStrikethrough     // ~~text~~ → "regexp"
	TokHighlight         // ==text== → "decorator"
	TokFootnote          // [^ref] → "label"
	TokMath              // $formula$ or $$block$$ → "operator"
	TokTableDelimiter    // | cell | separators → "struct"
	TokCallout           // > [!type] callout blocks → "typeParameter"
	TokEmbed             // ![[embed]] → "interface"
	TokHorizontalRule    // --- or *** → "enumMember"
	TokListMarker        // - or * or 1. → "enum"
)

// Token modifier bit flags.
const (
	ModDeclaration = 1 << iota // first occurrence / definition
	ModDefinition
	ModReadonly
	ModDeprecated   // e.g. completed task
	ModLink         // clickable reference
	ModAbstract     // referenced but not yet created
	ModStatic       // frontmatter / metadata
	ModAsync        // scheduled / future date
	ModModification // recently modified
	ModDefaultLibrary // built-in / template
)

var (
	trueVal  = true
	falseVal = false
	id       = "semanticTokens"

	// TokenTypes are the legend entries the client uses for theming.
	// Order must match the Tok* constants above.
	TokenTypes = []string{
		"namespace",      // heading
		"macro",          // tag
		"variable",       // mention
		"class",          // wiki link
		"function",       // md link
		"event",          // task
		"string",         // code span
		"number",         // date
		"property",       // frontmatter key
		"comment",        // blockquote
		"keyword",        // bold
		"modifier",       // italic
		"type",           // known entity
		"regexp",         // strikethrough
		"decorator",      // highlight
		"label",          // footnote
		"operator",       // math
		"struct",         // table delimiter
		"typeParameter",  // callout
		"interface",      // embed
		"enumMember",     // horizontal rule
		"enum",           // list marker
	}

	TokenModifiers = []string{
		"declaration",
		"definition",
		"readonly",
		"deprecated",
		"documentation",
		"abstract",
		"static",
		"async",
		"modification",
		"defaultLibrary",
	}

	workDone = protocol.WorkDoneProgressOptions{
		WorkDoneProgress: &trueVal,
	}

	Provider = protocol.SemanticTokensOptions{
		Full:                    true,
		Range:                   true,
		WorkDoneProgressOptions: workDone,
		Legend: protocol.SemanticTokensLegend{
			TokenTypes:     TokenTypes,
			TokenModifiers: TokenModifiers,
		},
	}

	Registration = protocol.SemanticTokensRegistrationOptions{
		TextDocumentRegistrationOptions: protocol.TextDocumentRegistrationOptions{
			DocumentSelector: &files.Filetypes,
		},
		SemanticTokensOptions: Provider,
		StaticRegistrationOptions: protocol.StaticRegistrationOptions{
			ID: &id,
		},
	}
)

// Token represents a single semantic token before delta-encoding.
type Token struct {
	Line      int
	StartChar int
	Length    int
	Type      int
	Modifiers int
}

// Encode converts a sorted slice of Tokens to the LSP delta-encoded uint array.
func Encode(tokens []Token) []protocol.UInteger {
	data := make([]protocol.UInteger, 0, len(tokens)*5)
	prevLine := 0
	prevChar := 0
	for _, tok := range tokens {
		deltaLine := tok.Line - prevLine
		deltaStart := tok.StartChar
		if deltaLine == 0 {
			deltaStart = tok.StartChar - prevChar
		}
		data = append(data,
			protocol.UInteger(deltaLine),
			protocol.UInteger(deltaStart),
			protocol.UInteger(tok.Length),
			protocol.UInteger(tok.Type),
			protocol.UInteger(tok.Modifiers),
		)
		prevLine = tok.Line
		prevChar = tok.StartChar
	}
	return data
}
