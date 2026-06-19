package context

import protocol "github.com/tliron/glsp/protocol_3_16"

type (
	Cursor struct {
		URI      protocol.DocumentUri
		position *protocol.Position

		InCodeFence  bool
		HeadingLevel int
		InHeading    bool
	}
)
