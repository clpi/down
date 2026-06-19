package util

import (
	"os"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func LnChar(f string, p protocol.Position) (string, string) {
	fl := strings.Split(f, "\n")
	ln := fl[p.Line]
	return ln, string(ln[p.Character])
}

func ReadLnChar(u protocol.URI, p protocol.Position) (string, string, error) {
	f, e := os.ReadFile(u)
	l, c := LnChar(string(f), p)
	return l, c, e
}
