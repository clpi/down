package link

import (
	"path"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	LinkTypeWiki = "wiki"
	LinkTypeMd   = "md"
	LinkTypeRef  = "ref"
)

type (
	LinkType    string
	LinkContext struct {
		Position protocol.Position
		File     string
	}
	RefLink struct {
		GetRef  func(r Ref) (Ref, error)
		Text    string
		Target  string
		Ref     string
		Context LinkContext
	}
	Link struct {
		Wiki struct {
			Target  string
			Context LinkContext
		}
		Md struct {
			Text    string
			Context LinkContext
		}
		Target string
		Loc    protocol.Position
		Text   string
		Path   protocol.URI
		Type   string
	}
	Ref struct {
		Ref     string
		Text    string
		Context LinkContext
		Links   func(r Ref) []Ref
	}
	WikiLink struct {
		Context LinkContext
		Text    string
	}
)

func (l LinkContext) Ext() string {
	return path.Ext(l.File)
}
func (l LinkContext) Target() string {
	return l.File
}
func (l1 WikiLink) Eq(l2 WikiLink) bool {
	return l2.Text == strings.Join([]string{l1.Text, l1.Context.Ext()}, ".")
}

func (l *WikiLink) Link() {

}

var ()
