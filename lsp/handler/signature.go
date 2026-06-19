package handler

import (
	"github.com/clpi/down/lsp/files"
	"github.com/clpi/down/lsp/handler/completion"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	Active            = protocol.UInteger(0)
	ActiveParameter   = protocol.UInteger(0)
	TriggerCharacters = []string{
		"@",
		// " ", "@", "#", "$", "%", "&",
	}
	RetriggerCharacters = []string{
		"@",
		// " ", "@", "#", "$", "%", "&",
	}
	workDone = protocol.WorkDoneProgressOptions{
		WorkDoneProgress: &t,
	}
	SignatureOptions = protocol.SignatureHelpOptions{
		WorkDoneProgressOptions: workDone,
		TriggerCharacters:       TriggerCharacters,
		RetriggerCharacters:     RetriggerCharacters,
	}
	Registration = protocol.CompletionRegistrationOptions{
		TextDocumentRegistrationOptions: files.DocumentRegistration,
		CompletionOptions:               completion.Provider,
	}
	SigParam = func(l, d string) protocol.ParameterInformation {
		return protocol.ParameterInformation{
			Label:         l,
			Documentation: d,
		}
	}
	SigInfo = func(l, d string, a int, p []protocol.ParameterInformation) protocol.SignatureInformation {
		ps := append(p, SigParam(l, d), SigParam("l1", "d1"))
		return protocol.SignatureInformation{
			Label:           l,
			Documentation:   d,
			ActiveParameter: &ActiveParameter,
			Parameters:      ps,
		}
	}
	Sig = func(p *protocol.SignatureHelpParams) *protocol.SignatureHelp {
		ss := []protocol.SignatureInformation{
			SigInfo("l2", "d2", 0, []protocol.ParameterInformation{}),
			SigInfo(
				"label",
				"# document\n\n# document\n\n+ info\n+ about\n+ document",
				0,
				[]protocol.ParameterInformation{
					SigParam("l1", "d1"),
					SigParam("l2", "d2"),
				},
			),
		}
		return &protocol.SignatureHelp{
			ActiveSignature: &Active,
			ActiveParameter: &ActiveParameter,
			Signatures:      ss,
		}
	}
)

func (s *State) SignatureHelp(c *glsp.Context, p *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return Sig(p), nil
}
