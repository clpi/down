package entities

import protocol "github.com/tliron/glsp/protocol_3_16"

const (
	LinkKindWiki     = "wiki"
	LinkKindFootnote = "footnote"
	LinkKindMarkdown = "markdown"
	LinkKindFile     = "file"
)
const (
	LinkUrlFtp   = "ftp://"
	LinkUrlHttp  = "http://"
	LinkUrlHttps = "https://"
	LinkTel      = "tel:"
	LinkSsh      = "ssh://"
	LinkSftp     = "sftp://"
	LinkFile     = "file://"
	LinkWiki     = "wiki://"
	LinkDown     = "down://"
	LinkEmail    = iota
)

type (
	Definition struct {
		Range  protocol.Range
		Target string
	}
	ListItem struct {
		Ln      protocol.UInteger
		Value   string
		level   int
		Ordered bool
	}
	Ref struct {
		Range  protocol.Range
		Target string
	}
	Link struct {
		Range  protocol.Range
		Value  string
		Target string
	}
	TagStr struct {
		Range protocol.Range
		Value string
	}
	Code struct {
		Range protocol.Range
		Value string
	}
	CodeBlock struct {
		Range    protocol.Range
		language string
		Value    string
	}
	Html struct {
		Range  protocol.Range
		Tag    string
		Value  string
		attrib []string
	}
	Header struct {
		Ln    protocol.UInteger
		Value string
		Level int
	}
	Math struct {
		Range protocol.Range
		Value string
	}
	WikiLink struct {
		Range protocol.Range
		Value string
	}
	Pointer struct {
		Range protocol.Range
		Value string
	}
	Footnote struct {
		Range protocol.Range
		Value string
	}
	Quote struct {
		Ln    protocol.UInteger
		Value string
		Level int
	}
	TaskStr struct {
		Ln    protocol.UInteger
		Value string
		Done  bool
	}
)
