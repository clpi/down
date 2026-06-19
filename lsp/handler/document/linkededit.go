package document

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func LinkedEditing(c *glsp.Context, p *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	var (
		p1 string                       = ""
		r1 protocol.LinkedEditingRanges = protocol.LinkedEditingRanges{
			WordPattern: &p1,
			Ranges: []protocol.Range{
				{
					Start: protocol.Position{
						Line:      10,
						Character: 10,
					},
					End: protocol.Position{
						Line:      20,
						Character: 20,
					},
				},
				{
					Start: protocol.Position{
						Line:      10,
						Character: 10,
					},
					End: protocol.Position{
						Line:      10,
						Character: 20,
					},
				},
			},
		}
	)
	return &r1, nil
}
