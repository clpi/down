package completion

import (
	"os"
	"path"
	"strings"

	"github.com/clpi/down/lsp/files"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var Edits = []protocol.TextDocumentEdit{}

type (
	LinkTrigger []string
	TriggerChar []string
	Completions []protocol.CompletionItem
)

var (
	/// "at ...", "by ",
	TriggerDate        = TriggerChar{"on ", "at ", "by ", "this ", "next ", "last "}
	TriggerList        = TriggerChar{"-", "+", "*", "<li>", ")", "."}
	TriggerFiles       = TriggerChar{"file://", "http://", "https://"}
	TriggerCode        = TriggerChar{"`"}
	TriggerCodeBlock   = TriggerChar{"```", "~~~"}
	TriggerBracketOpen = TriggerChar{"["}
	TriggerParenOpen   = TriggerChar{"("}
	TriggerCurlyOpen   = TriggerChar{"{"}
	TriggerHtml        = TriggerChar{"<"}
	TriggerQuote       = TriggerChar{">"}
	TriggerMath        = TriggerChar{"$"}
	TriggerCommant     = TriggerChar{"--", "<!--", "-->"}
	TriggerAt          = TriggerChar{"@"}
	TriggerRefDetail   = TriggerChar{"\"", "'", "("}
	TriggerLink        = TriggerChar{"["}
	TriggerLinkAuto    = TriggerChar{"<"}
	TriggerLinkTarget  = TriggerChar{"("}
	TriggerImage       = TriggerChar{"!"}
	TriggerTable       = TriggerChar{"|"}
	TriggerDefinition  = TriggerChar{":"}
	TriggerRef         = TriggerChar{"["}
	TriggerTag         = TriggerChar{"#"}
	TriggerHeader      = TriggerChar{"#"}
	TriggerTask        = TriggerChar{"["}
	TriggerVariable    = TriggerChar{"&"}
)

type CompletionTrigger struct {
	Kind     protocol.CompletionTriggerKind
	Position protocol.Position
	Body     []byte
	Char     *string
}

func Trigger(p *protocol.CompletionParams) *CompletionTrigger {
	f, e := os.ReadFile(path.Clean(p.TextDocument.URI))
	if e != nil {
		return nil
	}
	return &CompletionTrigger{
		Kind:     p.Context.TriggerKind,
		Char:     p.Context.TriggerCharacter,
		Position: p.Position,
		Body:     f,
	}
}

func (c *CompletionTrigger) Line() string {
	return strings.Split(string(c.Body), "\n")[c.Position.Line]
}

func (c *CompletionTrigger) SplitLn() []string {
	return strings.Split(c.Line(), " ")
}

func (c *CompletionTrigger) Next() byte {
	return c.Line()[c.Position.Character+1]
}

func (c *CompletionTrigger) Prev() byte {
	return c.Line()[c.Position.Character-1]
}

func (c *CompletionTrigger) MatchChar() []string {
	switch *c.Char {
	case "-":
	case "*":
	case "/":
	case "+":
	default:

	}
	return strings.Split(string(c.Body), " ")
}

func (c *CompletionTrigger) MatchKind() Completions {
	out := Completions{}
	switch c.Kind {
	case protocol.CompletionTriggerKindTriggerCharacter:
		return out
	case protocol.CompletionTriggerKindTriggerForIncompleteCompletions:
		return out
	case protocol.CompletionTriggerKindInvoked:
		return out
	}
	return out
}

func Match(p *protocol.CompletionParams) Completions {
	var ct *CompletionTrigger = Trigger(p)
	return ct.MatchKind()
}

var (
	t                                   = true
	f                                   = false
	Provider protocol.CompletionOptions = protocol.CompletionOptions{
		ResolveProvider: &t,
		WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
			WorkDoneProgress: &t,
		},
		AllCommitCharacters: []string{
			" ", "@", "#", "$", "%", "&",
			"*", "+", "-", "/", "<", "=",
		},
		TriggerCharacters: []string{
			"@", "#", "$", "%", "&",
			"*", "+", "-", "/", "<", "=",
			">", "?", "^", "|", "~",
			"[", "(", "<", "{", "`",
			"]", ")", ">", "}",
			":", "=", ",",
			".", ";", "'",
			"\"", "'", "\\", "/",
			"!", "_",
			"~", "`",
		},
	}
	p        protocol.CompletionItemTag
	Register = protocol.CompletionRegistrationOptions{
		CompletionOptions:               Provider,
		TextDocumentRegistrationOptions: files.DocumentRegistration,
	}
)
