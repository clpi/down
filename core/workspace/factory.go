package workspace

import protocol "github.com/tliron/glsp/protocol_3_16"

type (
	Factory interface {
		Invoke()
		Create(name string, uri string) (*Workspace, error)
		List() []*Workspace
	}
)

type (
	AddNoteFunc func(name string, dir protocol.URI, template string) error
)

// WorkspaceFactory provides workspace creation and management.
type WorkspaceFactory struct {
	manager *Workspaces
}

// NewFactory creates a workspace factory with a manager.
func NewFactory(manager *Workspaces) *WorkspaceFactory {
	return &WorkspaceFactory{manager: manager}
}

func (f *WorkspaceFactory) Invoke() {}

func (f *WorkspaceFactory) Create(name string, uri string) (*Workspace, error) {
	w := NewWorkspace(name, uri)
	f.manager.Add(w)
	return w, nil
}

func (f *WorkspaceFactory) List() []*Workspace {
	return f.manager.All()
}
