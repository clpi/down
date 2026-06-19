package handler

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/clpi/down/lsp/ai"
	"github.com/clpi/down/lsp/handler/completion"
	"github.com/clpi/down/lsp/handler/semantic"
	"github.com/clpi/down/lsp/knowledge"
)

var (
	name    string = "down"
	version string = "0.1.0-alpha"
	ph      protocol.Handler
)

type Handler struct {
	protocol.Handler
	initialized bool
	lock        sync.Mutex
}

func Capabilities() protocol.ServerCapabilities {
	cb := ph.CreateServerCapabilities()
	cb.CompletionProvider = &completion.Provider
	cb.CodeActionProvider = &ActionProvider
	cb.CodeLensProvider = &LensProvider
	cb.ExecuteCommandProvider = &CommandProvider
	cb.DocumentLinkProvider = &DocumentProvider.Link
	cb.DocumentHighlightProvider = &DocumentProvider.Highlight
	cb.HoverProvider = &DocumentProvider.Hover
	cb.DefinitionProvider = &DocumentProvider.Definition
	cb.ReferencesProvider = true
	cb.DocumentSymbolProvider = &DocumentProvider.Symbol
	cb.TextDocumentSync = &DocumentProvider.Sync
	cb.SemanticTokensProvider = &semantic.Provider
	cb.RenameProvider = true
	cb.DocumentFormattingProvider = true
	cb.DocumentRangeFormattingProvider = true
	cb.DocumentOnTypeFormattingProvider = &protocol.DocumentOnTypeFormattingOptions{
		FirstTriggerCharacter: "\n",
		MoreTriggerCharacter:  []string{},
	}
	cb.FoldingRangeProvider = true
	cb.SelectionRangeProvider = true
	cb.LinkedEditingRangeProvider = true
	cb.Workspace = &protocol.ServerCapabilitiesWorkspace{
		WorkspaceFolders: &protocol.WorkspaceFoldersServerCapabilities{},
		FileOperations:   &WorkspaceFilesProvider,
	}
	return cb
}

func ServerInfo() *protocol.InitializeResultServerInfo {
	return &protocol.InitializeResultServerInfo{
		Name:    name,
		Version: &version,
	}
}

func (s *State) Shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	commonlog.NewInfoMessage(0, "down shutdown...")
	return nil
}

func (s *State) SetTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func (s *State) Cancel(c *glsp.Context, p *protocol.CancelParams) error {
	return nil
}

func (s *State) LogTrace(c *glsp.Context, p *protocol.LogTraceParams) error {
	if p.Verbose != nil {
		log.Print(p.Message, c.Method)
	} else {
		log.Print(p.Message, c.Method)
	}
	return nil
}

func (s *State) Initialized(c *glsp.Context, p *protocol.InitializedParams) error {
	protocol.SetTraceValue(protocol.TraceValueVerbose)
	return nil
}

func (s State) Handlers() protocol.Handler {
	return protocol.Handler{
		Initialize:                     s.Initialize,
		Initialized:                    s.Initialized,
		LogTrace:                       s.LogTrace,
		SetTrace:                       s.SetTrace,
		Shutdown:                       s.Shutdown,
		Exit:                           s.Exit,
		TextDocumentLinkedEditingRange: s.LinkedEditing,
		TextDocumentDocumentHighlight:  s.DocumentHighlight,
		TextDocumentDocumentLink:       s.Links,
		TextDocumentCodeLens:           s.CodeLens,
		TextDocumentSemanticTokensFullDelta: s.Delta,
		WorkspaceSemanticTokensRefresh:      s.Refresh,
		TextDocumentSemanticTokensFull:      s.Full,
		TextDocumentSemanticTokensRange:     s.Range,
		// TextDocumentSignatureHelp:           s.SignatureHelp,

		WorkspaceDidChangeConfiguration: s.Configure,
		WorkspaceDidChangeWatchedFiles:  s.WsDidWatch,
		WorkspaceDidCreateFiles:         s.WsDidCreate,
		WorkspaceDidDeleteFiles:         s.WsDidDelete,
		WorkspaceDidRenameFiles:         s.WsDidRename,
		WorkspaceWillCreateFiles:        s.WsWillCreate,
		WorkspaceWillDeleteFiles:        s.WsDelete,
		WorkspaceWillRenameFiles:        s.WsRename,
		WorkspaceExecuteCommand:         s.Command,
		DocumentLinkResolve:             s.LinkResolve,
		TextDocumentCodeAction:          s.CodeAction,
		TextDocumentRename:              s.Rename,
		TextDocumentPrepareRename:       s.PrepareRename,
		TextDocumentColor:               s.Color,
		TextDocumentDocumentSymbol:      s.Symbol,
		WorkspaceSymbol:                 s.WorkspaceSymbol,
		Progress:                        s.Progress,

		WorkspaceDidChangeWorkspaceFolders: s.ChangeWorkspaceFolders,
		TextDocumentMoniker:                s.Moniker,
		CompletionItemResolve:              s.ItemResolve,
		CodeActionResolve:                  s.ActionResolve,
		TextDocumentColorPresentation:      s.ColorPresentation,
		CodeLensResolve:                    s.LensResolve,
		TextDocumentCompletion:             s.Completion,
		TextDocumentHover:                  s.Hover,
		TextDocumentReferences:             s.References,
		WindowWorkDoneProgressCancel:       s.CancelWorkDoneProgresss,
		TextDocumentDidOpen:                s.DidOpen,
		TextDocumentDidChange:              s.DidChange,

		TextDocumentDidClose:          s.DidClose,
		TextDocumentWillSave:          s.WillSave,
		TextDocumentWillSaveWaitUntil: s.WillSaveWaitUntil,
		TextDocumentDidSave:           s.DidSave,
		CancelRequest:                 s.Cancel,
		// TextDocumentSignatureHelp:          s.SignatureHelp,

		// TextDocumentTypeDefinition:          document.TypeDefinition,
		// TextDocumentSemanticTokensFullDelta: semantic.Delta,
		// TextDocumentSemanticTokensFull:      semantic.Full,
		// TextDocumentSemanticTokensRange:     semantic.Range,
		// WorkspaceSemanticTokensRefresh:      semantic.Workspace,
		// TextDocumentDeclaration:            document.Declaration,
		TextDocumentDefinition:             s.Definition,
		TextDocumentFormatting:             s.Format,
		TextDocumentRangeFormatting:        s.RangeFormat,
		TextDocumentOnTypeFormatting:       s.OnTypeFormatting,
		TextDocumentFoldingRange:           s.FoldingRange,
		TextDocumentSelectionRange:         s.SelectionRange,
	}
}

// Title: "Link current word to new file in workspace",
// Disabled:    &falseVal,
// IsPreferred: &trueVal,
// NewText: "[text](./text.md)",

func NewState() State {
	home, _ := os.UserHomeDir()
	storePath := filepath.Join(home, ".down", "knowledge.json")
	graph := knowledge.NewGraph(storePath)

	var engine *ai.Engine
	provider, err := ai.SelectProvider()
	if err != nil {
		log.Printf("AI provider: %v (completions disabled, knowledge graph still active)", err)
	} else {
		log.Printf("AI provider: %s", provider.Name())
		engine = ai.NewEngine(provider, graph)
	}

	return State{
		Graph:     graph,
		AI:        engine,
		Documents: make(map[string]string),
	}
}

func (s *State) Exit(context *glsp.Context) error {
	commonlog.NewInfoMessage(0, "down exit...")
	if s.Graph != nil {
		s.Graph.Save()
	}
	return nil
}

func (s *State) Initialize(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
	commonlog.NewInfoMessage(0, "down init...")
	protocol.SetTraceValue(protocol.TraceValueVerbose)

	if params.WorkspaceFolders != nil && s.Graph != nil {
		go func() {
			var roots []string
			for _, f := range params.WorkspaceFolders {
				roots = append(roots, f.URI)
			}
			n := knowledge.ScanWorkspace(s.Graph, roots)
			log.Printf("Scanned %d markdown files into knowledge graph", n)
		}()
	} else if params.RootURI != nil && s.Graph != nil {
		go func() {
			n := knowledge.ScanWorkspace(s.Graph, []string{string(*params.RootURI)})
			log.Printf("Scanned %d markdown files into knowledge graph", n)
		}()
	}

	return protocol.InitializeResult{
		Capabilities: Capabilities(),
		ServerInfo:   ServerInfo(),
	}, nil
}

func (s *State) Progress(context *glsp.Context, params *protocol.ProgressParams) error {
	return nil
}

func (s *State) CancelWorkDoneProgresss(context *glsp.Context, params *protocol.WorkDoneProgressCancelParams) error {
	return nil
}
