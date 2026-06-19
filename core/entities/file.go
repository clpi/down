package entities

import protocol "github.com/tliron/glsp/protocol_3_16"

type (
	Path interface {
		string
		string
		protocol.URI
		[]Tag
		id() string
		uri() protocol.URI
		tags() []Tag
		name() string
	}
	File struct {
		Id   string
		Name string
		Uri  protocol.URI
		Tags []Tag
	}
	Dir struct {
		Id   string
		Uri  protocol.URI
		Tags []Tag
		Name string
	}
	Channel chan string
)

func (f *File) Dir() Dir {
	return Dir{
		Id:  f.Id,
		Uri: f.Uri,
	}
}
func (f *File) Workspace() string {
	return "workspace"
}
