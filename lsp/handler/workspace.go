package handler

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/lsp/files"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var WorkspaceFilesProvider = protocol.ServerCapabilitiesWorkspaceFileOperations{
	DidCreate: &protocol.FileOperationRegistrationOptions{
		Filters: files.FileOps,
	},
	WillCreate: &protocol.FileOperationRegistrationOptions{
		Filters: files.FileOps,
	},
	DidRename: &protocol.FileOperationRegistrationOptions{
		Filters: files.FileOps,
	},
	WillRename: &protocol.FileOperationRegistrationOptions{
		Filters: files.FileOps,
	},
	WillDelete: &protocol.FileOperationRegistrationOptions{
		Filters: files.FileOps,
	},
	DidDelete: &protocol.FileOperationRegistrationOptions{
		Filters: files.FileOps,
	},
}

//   "workspace.workspace_folders": []protocol.WorkspaceFolder
// }

// func (s *State) WsCreate(c *glsp.Context, p *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
// 	return nil, nil
// }

func (s *State) WsDidCreate(_ *glsp.Context, p *protocol.CreateFilesParams) error {
	if s.Graph == nil {
		return nil
	}
	for _, f := range p.Files {
		uri := f.URI
		s.scanFileIntoGraph(uri)
	}
	return nil
}

func (s *State) WsDelete(_ *glsp.Context, _ *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *State) WsDidDelete(_ *glsp.Context, p *protocol.DeleteFilesParams) error {
	if s.Graph == nil {
		return nil
	}
	for _, f := range p.Files {
		s.Graph.ClearDocument(f.URI)
	}
	s.Graph.Save()
	return nil
}

func (s *State) WsWatch(_ *glsp.Context, _ *protocol.DidChangeWatchedFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *State) WsDidWatch(_ *glsp.Context, p *protocol.DidChangeWatchedFilesParams) error {
	if s.Graph == nil {
		return nil
	}
	for _, change := range p.Changes {
		uri := string(change.URI)
		switch change.Type {
		case protocol.FileChangeTypeCreated, protocol.FileChangeTypeChanged:
			s.scanFileIntoGraph(uri)
		case protocol.FileChangeTypeDeleted:
			s.Graph.ClearDocument(uri)
			delete(s.Documents, uri)
		}
	}
	s.Graph.Save()
	return nil
}

func (s *State) WsRename(_ *glsp.Context, _ *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *State) WsDidRename(_ *glsp.Context, p *protocol.RenameFilesParams) error {
	if s.Graph == nil {
		return nil
	}
	for _, f := range p.Files {
		s.Graph.ClearDocument(f.OldURI)
		delete(s.Documents, f.OldURI)
		s.scanFileIntoGraph(f.NewURI)
	}
	s.Graph.Save()
	return nil
}

func (s *State) WsWillCreate(_ *glsp.Context, _ *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (s *State) scanFileIntoGraph(uri string) {
	path := strings.TrimPrefix(uri, "file://")
	path = strings.TrimPrefix(path, "file:")
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" && ext != ".markdown" && ext != ".mdx" && ext != ".txt" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(data)
	s.Documents[uri] = text
	knowledge.ExtractFromDocument(s.Graph, uri, text)
}

func (s *State) Configure(_ *glsp.Context, _ *protocol.DidChangeConfigurationParams) error {
	return nil
}

func (s *State) WorkspaceSymbol(_ *glsp.Context, p *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	query := strings.TrimSpace(p.Query)
	if query == "" {
		return []protocol.SymbolInformation{}, nil
	}

	seen := make(map[string]bool)
	symbols := make([]protocol.SymbolInformation, 0)

	kindMap := map[knowledge.EntityKind]protocol.SymbolKind{
		knowledge.KindPerson:   protocol.SymbolKindVariable,
		knowledge.KindConcept:  protocol.SymbolKindClass,
		knowledge.KindProject:  protocol.SymbolKindPackage,
		knowledge.KindAction:   protocol.SymbolKindFunction,
		knowledge.KindTag:      protocol.SymbolKindKey,
		knowledge.KindDocument: protocol.SymbolKindFile,
		knowledge.KindDate:     protocol.SymbolKindEvent,
		knowledge.KindPlace:    protocol.SymbolKindNamespace,
		knowledge.KindCode:     protocol.SymbolKindObject,
	}

	if s.Graph != nil {
		for _, ent := range s.Graph.Search(query) {
			kind, ok := kindMap[ent.Kind]
			if !ok {
				kind = protocol.SymbolKindString
			}
			for _, src := range ent.Sources {
				key := string(src.URI) + ":" + ent.Name
				if seen[key] {
					continue
				}
				seen[key] = true
				symbols = append(symbols, protocol.SymbolInformation{
					Name: ent.Name,
					Kind: kind,
					Location: protocol.Location{
						URI: protocol.DocumentUri(src.URI),
						Range: protocol.Range{
							Start: protocol.Position{Line: protocol.UInteger(src.Line), Character: 0},
							End:   protocol.Position{Line: protocol.UInteger(src.Line), Character: protocol.UInteger(len(ent.Name))},
						},
					},
				})
				break
			}
		}
	}

	lower := strings.ToLower(query)
	for uri, text := range s.Documents {
		title := getDocTitle(s.Documents, uri)
		if title != "" && strings.Contains(strings.ToLower(title), lower) {
			key := uri + ":doc:" + title
			if !seen[key] {
				seen[key] = true
				symbols = append(symbols, protocol.SymbolInformation{
					Name: title,
					Kind: protocol.SymbolKindFile,
					Location: protocol.Location{
						URI: protocol.DocumentUri(uri),
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: protocol.UInteger(len(title))},
						},
					},
				})
			}
		}

		lines := strings.Split(text, "\n")
		for i, line := range lines {
			if m := reTask.FindStringSubmatch(line); m != nil {
				taskText := strings.TrimSpace(m[2])
				if strings.Contains(strings.ToLower(taskText), lower) {
					key := uri + ":task:" + intStr(i)
					if seen[key] {
						continue
					}
					seen[key] = true
					symbols = append(symbols, protocol.SymbolInformation{
						Name: taskText,
						Kind: protocol.SymbolKindEvent,
						Location: protocol.Location{
							URI: protocol.DocumentUri(uri),
							Range: protocol.Range{
								Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
								End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
							},
						},
					})
				}
			}

			for _, m := range reLinkTag.FindAllStringSubmatchIndex(line, -1) {
				tag := line[m[2]:m[3]]
				if strings.Contains(strings.ToLower(tag), lower) {
					key := uri + ":tag:" + tag
					if seen[key] {
						continue
					}
					seen[key] = true
					symbols = append(symbols, protocol.SymbolInformation{
						Name: tag,
						Kind: protocol.SymbolKindKey,
						Location: protocol.Location{
							URI: protocol.DocumentUri(uri),
							Range: protocol.Range{
								Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(m[2])},
								End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(m[3])},
							},
						},
					})
				}
			}
		}
	}

	return symbols, nil
}

func (s *State) ChangeWorkspaceFolders(c *glsp.Context, p *protocol.DidChangeWorkspaceFoldersParams) error {
	if s.Graph == nil {
		return nil
	}

	// Handle added folders
	for _, folder := range p.Event.Added {
		go func(uri string) {
			n := ScanWorkspace(s.Graph, []string{uri})
			_ = n
		}(folder.URI)
	}

	// Handle removed folders
	for _, folder := range p.Event.Removed {
		entities := s.Graph.EntitiesByDocument(folder.URI)
		for _, ent := range entities {
			for _, src := range ent.Sources {
				if strings.HasPrefix(src.URI, folder.URI) {
					s.Graph.ClearDocument(src.URI)
				}
			}
		}
	}

	s.Graph.Save()
	return nil
}

// ScanWorkspace is a helper for scanning workspace folders.
func ScanWorkspace(g *knowledge.Graph, roots []string) int {
	return knowledge.ScanWorkspace(g, roots)
}
