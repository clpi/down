package link

import (
	"github.com/clpi/down/internal/document"
	// protocol "github.com/tliron/glsp/protocol_3_16"
)

type (
	Links struct {
		Links     map[document.DocumentID]*document.Document
		Backlinks map[document.DocumentID][]document.DocumentID
		ID        map[string]document.DocumentID
	}
)

func p() {
}
