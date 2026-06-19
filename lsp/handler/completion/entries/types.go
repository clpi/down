package entries

import protocol "github.com/tliron/glsp/protocol_3_16"

var (
	f = false
	t = true
)

type (
	Template struct {
		Body        string               `json:"body",yaml:"body"`
		Description string               `json:"description",yaml:"description"`
		Document    protocol.DocumentUri `json:"document",yaml:"document"`
		Workspace   string               `json:"workspace",yaml:"workspace"`
		URI         protocol.URI         `json:"uri",yaml:"uri"`
	}
	Log struct {
		Body string
	}
	Note struct {
		Body string
	}
	Document struct {
	}
	Snippet struct {
		Body        string
		Description string
	}
	Workspace struct {
		Name    string
		Path    protocol.URI
		Default bool
		Index   string
		Notes   string
		Config  interface{}
	}
)
