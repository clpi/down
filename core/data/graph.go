package data

import (
	"time"

	"github.com/clpi/down/core/entities"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type (
	Target struct {
		Value    string
		Uri      protocol.URI
		Protocol string
		Anchor   string
	}
	File struct {
		Uri       protocol.URI
		Data      Data
		Anchors   []string
		Links     []Link
		Backlinks []Link
	}
	Link struct {
		Target   Target
		LinkType int
		Exists   bool
		Position protocol.Position
		Data     Data
	}
	Data struct {
		Id      string
		Text    string
		About   string
		Hidden  bool
		Tags    []entities.Tag
		Created time.Time
	}
	Workspace struct {
		Data      Data
		Root      protocol.URI
		Index     string
		Notes     string
		Logs      string
		Snippets  string
		Agenda    string
		Tasks     string
		Templates string
		Files     []File
		Links     []Link
	}
	InterWorkspace struct {
		Data       Data
		Profile    string
		Workspaces []Workspace
	}
	Path struct {
		Base string
		Ext  string
		Dir  string
	}
)
