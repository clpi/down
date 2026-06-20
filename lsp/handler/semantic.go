package handler

import (
	"strings"

	"github.com/clpi/down/lsp/handler/semantic"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *State) knownEntitySet() map[string]bool {
	if s.Graph == nil {
		return nil
	}
	entities := s.Graph.AllEntities()
	if len(entities) == 0 {
		return nil
	}
	known := make(map[string]bool, len(entities)*2)
	for _, ent := range entities {
		if ent.Name == "" {
			continue
		}
		known[ent.Name] = true
		known[strings.ToLower(ent.Name)] = true
	}
	return known
}

func (s *State) semanticTokensFor(uri, text string) []semantic.Token {
	return semantic.TokenizeWithGraph(text, s.knownEntitySet())
}

func (s *State) Full(_ *glsp.Context, p *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	tokens := s.semanticTokensFor(uri, text)
	if len(tokens) == 0 {
		return nil, nil
	}

	data := semantic.Encode(tokens)
	return &protocol.SemanticTokens{Data: data}, nil
}

func (s *State) Delta(_ *glsp.Context, _ *protocol.SemanticTokensDeltaParams) (any, error) {
	// For now, return nil to let the client fall back to a full request.
	return nil, nil
}

func (s *State) Refresh(_ *glsp.Context) error {
	return nil
}

func (s *State) Range(_ *glsp.Context, p *protocol.SemanticTokensRangeParams) (any, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	allTokens := s.semanticTokensFor(uri, text)
	startLine := int(p.Range.Start.Line)
	endLine := int(p.Range.End.Line)

	var rangeTokens []semantic.Token
	for _, tok := range allTokens {
		if tok.Line >= startLine && tok.Line <= endLine {
			rangeTokens = append(rangeTokens, tok)
		}
	}

	if len(rangeTokens) == 0 {
		return nil, nil
	}

	data := semantic.Encode(rangeTokens)
	return &protocol.SemanticTokens{Data: data}, nil
}
