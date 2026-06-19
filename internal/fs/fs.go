package fs

import (
	"bytes"
	// "filepath"
	// "fs"
	iofs "io/fs"
	// "time"
)

type InMem struct {
	bytes []byte
	info  iofs.FileInfo
}

func (m *InMem) Read(f []byte) (int, error) {
	return bytes.NewBuffer(m.bytes).Read(f)
}
