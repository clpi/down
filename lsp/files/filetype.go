package files

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	trueVal  = true
	falseVal = false
	file     = "file"
)

func Filetype(f string, p string, s string) protocol.DocumentFilter {
	return protocol.DocumentFilter{
		Language: &f,
		Scheme:   &s,
		Pattern:  &p,
	}
}

func FileOp(f string, s string) protocol.FileOperationFilter {
	return protocol.FileOperationFilter{
		Scheme: &file,
		Pattern: protocol.FileOperationPattern{
			Glob: s,
			Options: &protocol.FileOperationPatternOptions{
				IgnoreCase: &trueVal,
			},
		},
	}

}

var (
	Filetypes = protocol.DocumentSelector{
		Filetype("down", "**/*.down", file),
		Filetype("docdown", "**/*.docdown", file),
		Filetype("markdown", "**/*.markdown", file),
	}
	FileOps = []protocol.FileOperationFilter{
		FileOp("down", "**/*.down"),
		FileOp("docdown", "**/*.docdown"),
		FileOp("markdown", "**/*.markdown"),
	}
)

const (
	ExtMdx      = "mdx"
	ExtMarkdown = "md"
	ExtDown     = "dn"
	ExtDocdown  = "dd"
	ExtOther
)

type (
	Extensions     []string
	DownExtensions struct {
		Mdx      Extensions
		Markdown Extensions
		Down     Extensions
		Docdown  Extensions
	}
)

var (
	DownExt = DownExtensions{
		Mdx: Extensions{
			"md",
		},
		Markdown: Extensions{
			"mdx",
		},
		Down: Extensions{
			"dn",
			"down",
			"dwn",
			"dw",
			"do",
		},
		Docdown: Extensions{
			"dd",
			"ddn",
			"ddo",
			"ddoc",
			"docd",
		},
	}
	DocumentRegistration = protocol.TextDocumentRegistrationOptions{
		DocumentSelector: &Filetypes,
	}
)

func Ext(uri protocol.URI) string {
	sp := strings.Split(uri, ".")
	return sp[len(sp)-1]
}
func IsMarkdown(uri protocol.URI) bool {
	e := Ext(uri)
	return e == ExtMarkdown || e == ExtMdx
}
func IsDownFile(uri protocol.URI) bool {
	e := Ext(uri)
	return (e == ExtDown || e == ExtDocdown)
}
