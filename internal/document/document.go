package document

import (
	"path/filepath"
	"time"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Data map[string]interface{}

type (
	DocumentID string
	Document   struct {
		ID          DocumentID
		WorkspaceID string
		URI         protocol.URI
		Title       string
		Lead        string
		Body        string
		Metadata    map[string]interface{}
		Links       []interface{}
		Tags        []string
		Created     time.Time
		Updated     time.Time
	}
)

// TODO: Way to have workspace file that does not exist in folder children
func Base(d Document) string {
	return filepath.Base(d.URI)
}

func Dir(d Document) string {
	return filepath.Dir(d.URI)
}

func FromFile(s []byte) Document {
	d := Document{}
	return d
}

func FromURI(p string) (Document, error) {
	d := Document{}
	return d, nil
}
