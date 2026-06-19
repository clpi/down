package util

import protocol "github.com/tliron/glsp/protocol_3_16"

func Pos(l protocol.UInteger, c protocol.UInteger) protocol.Position {
	return protocol.Position{
		Line:      l,
		Character: c,
	}
}
func Rng(s protocol.Position, e protocol.Position) protocol.Range {
	return protocol.Range{
		Start: s,
		End:   e,
	}
}
func Range(l1 protocol.UInteger, c1 protocol.UInteger, l2 protocol.UInteger, c2 protocol.UInteger) protocol.Range {
	return protocol.Range{
		Start: Pos(l1, c1),
		End:   Pos(l2, c2),
	}
}

func Top() protocol.Position {
	return Pos(0, 0)
}
func Start(l protocol.UInteger) protocol.Position {
	return Pos(l, 0)
}

func Delta(l protocol.UInteger, dl protocol.UInteger, c protocol.UInteger, dc protocol.UInteger) protocol.Position {
	return Pos(l+dl, c+dc)
}
func Dl(l protocol.UInteger, dl protocol.UInteger, c protocol.UInteger) protocol.Position {
	return Delta(l, dl, c, 0)
}
func Dc(l protocol.UInteger, c protocol.UInteger, dc protocol.UInteger) protocol.Position {
	return Delta(l, 0, c, dc)
}
func NextCol(l protocol.UInteger, c protocol.UInteger) protocol.Position {
	return Dc(l, c, 1)
}
func NextLine(l protocol.UInteger, c protocol.UInteger) protocol.Position {
	return Dl(l, 1, c)
}

func ToNextLineWithCol(l protocol.UInteger, c1 protocol.UInteger, c2 protocol.UInteger) protocol.Range {
	return Range(l, c1, l+1, c2)
}
func ToPrevChar(l protocol.UInteger, c protocol.UInteger) protocol.Range {
	return Range(l, max(c-1, 0), l, c)
}
func ToNextChar(l protocol.UInteger, c protocol.UInteger) protocol.Range {
	return Range(l, c, l, c+1)
}
func ToPrevLine(l protocol.UInteger, c protocol.UInteger) protocol.Range {
	return Range(max(l-1, 0), c, l, c)
}
func ToCol(l protocol.UInteger, c protocol.UInteger) protocol.Range {
	return Range(l, 0, l, c)
}
func ToNextLine(l protocol.UInteger, c protocol.UInteger) protocol.Range {
	return Range(l, c, l+1, 0)
}
func WholeLine(l protocol.UInteger) protocol.Range {
	return Range(l, 0, l+1, 0)
}
func ToTop(l protocol.UInteger, c protocol.UInteger) protocol.Range {
	return Range(0, 0, l, c)
}
