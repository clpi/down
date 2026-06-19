package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	store "github.com/clpi/down/core/store"
	_ "go.lsp.dev/protocol"
)

type (
	WorkspaceConfig = map[string]interface{}
)

var (
	DefaultWorkspaceConfig = map[string]interface{}{
		"workspace": map[string]interface{}{
			"indexName": "index.md",
			"directories": map[string]interface{}{
				"store": map[string]interface{}{
					"default": ".down",
				},
				"notes": map[string]interface{}{
					"default": "notes",
				},
				"templates": map[string]interface{}{
					"default": "templates",
				},
				"snippets": map[string]interface{}{
					"default": "snippets",
				},
				"journal": map[string]interface{}{
					"default": "journal",
				},
				"attachments": map[string]interface{}{
					"default": "attachments",
				},
			},
			"knowledge": map[string]interface{}{
				"autoIndex":    true,
				"indexOnOpen":  true,
				"graphStorage": "json",
			},
			"ai": map[string]interface{}{
				"enabled":     true,
				"provider":    "auto",
				"completions": true,
			},
		},
	}
)

type (
	Name       string
	Identifier string
)

// WorkspaceState represents the runtime state of a workspace.
type WorkspaceState int

const (
	WorkspaceStateUninitialized WorkspaceState = iota
	WorkspaceStateInitializing
	WorkspaceStateReady
	WorkspaceStateError
)

// WorkspaceEvent types for workspace lifecycle.
type WorkspaceEventKind string

const (
	EventWorkspaceCreated     WorkspaceEventKind = "created"
	EventWorkspaceOpened      WorkspaceEventKind = "opened"
	EventWorkspaceClosed      WorkspaceEventKind = "closed"
	EventWorkspaceFileChanged WorkspaceEventKind = "file_changed"
	EventWorkspaceFileCreated WorkspaceEventKind = "file_created"
	EventWorkspaceFileDeleted WorkspaceEventKind = "file_deleted"
	EventWorkspaceReindexed   WorkspaceEventKind = "reindexed"
	EventWorkspaceConfigured  WorkspaceEventKind = "configured"
)

// WorkspaceEvent records something that happened in a workspace.
type WorkspaceEvent struct {
	Kind      WorkspaceEventKind `json:"kind"`
	URI       string             `json:"uri,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
	Data      interface{}        `json:"data,omitempty"`
}

// WorkspaceSettings holds user-configurable settings for a workspace.
type WorkspaceSettings struct {
	IndexName     string            `json:"index_name"`
	NotesDir      string            `json:"notes_dir"`
	TemplatesDir  string            `json:"templates_dir"`
	SnippetsDir   string            `json:"snippets_dir"`
	JournalDir    string            `json:"journal_dir"`
	AttachmentsDir string           `json:"attachments_dir"`
	StoreDir      string            `json:"store_dir"`
	ExcludeGlobs  []string          `json:"exclude_globs,omitempty"`
	IncludeGlobs  []string          `json:"include_globs,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	AIEnabled     bool              `json:"ai_enabled"`
	AutoIndex     bool              `json:"auto_index"`
	CustomMeta    map[string]string `json:"custom_meta,omitempty"`
}

// DefaultSettings returns workspace settings with sensible defaults.
func DefaultSettings() WorkspaceSettings {
	return WorkspaceSettings{
		IndexName:      "index.md",
		NotesDir:       "notes",
		TemplatesDir:   "templates",
		SnippetsDir:    "snippets",
		JournalDir:     "journal",
		AttachmentsDir: "attachments",
		StoreDir:       ".down",
		ExcludeGlobs:   []string{".git", "node_modules", ".obsidian", ".trash"},
		IncludeGlobs:   []string{"*.md", "*.markdown", "*.mdx", "*.txt"},
		Tags:           make(map[string]string),
		AIEnabled:      true,
		AutoIndex:      true,
		CustomMeta:     make(map[string]string),
	}
}

// Workspace represents a managed workspace with full lifecycle support.
type Workspace struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	URI       string             `json:"uri"`
	State     WorkspaceState     `json:"state"`
	Settings  WorkspaceSettings  `json:"settings"`
	CreatedAt time.Time          `json:"created_at"`
	OpenedAt  time.Time          `json:"opened_at,omitempty"`
	FileCount int                `json:"file_count"`
	Events    []WorkspaceEvent   `json:"events,omitempty"`
	mu        sync.RWMutex
}

// Workspaces manages multiple workspaces concurrently.
type Workspaces struct {
	workspaces map[string]*Workspace
	active     string
	mu         sync.RWMutex
	listeners  []func(WorkspaceEvent)
}

// NewWorkspaces creates a new multi-workspace manager.
func NewWorkspaces() *Workspaces {
	return &Workspaces{
		workspaces: make(map[string]*Workspace),
		listeners:  make([]func(WorkspaceEvent), 0),
	}
}

// NewWorkspace creates a new workspace with the given ID and URI.
func NewWorkspace(id string, uri string) *Workspace {
	return &Workspace{
		ID:        id,
		Name:      filepath.Base(uri),
		URI:       uri,
		State:     WorkspaceStateUninitialized,
		Settings:  DefaultSettings(),
		CreatedAt: time.Now(),
		Events:    make([]WorkspaceEvent, 0),
	}
}

// Add adds a workspace to the manager.
func (ws *Workspaces) Add(w *Workspace) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.workspaces[w.ID] = w
	if ws.active == "" {
		ws.active = w.ID
	}
	ws.emit(WorkspaceEvent{
		Kind:      EventWorkspaceCreated,
		URI:       w.URI,
		Timestamp: time.Now(),
	})
}

// Remove removes a workspace by ID.
func (ws *Workspaces) Remove(id string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if _, ok := ws.workspaces[id]; !ok {
		return fmt.Errorf("workspace %q not found", id)
	}
	delete(ws.workspaces, id)
	if ws.active == id {
		ws.active = ""
		for k := range ws.workspaces {
			ws.active = k
			break
		}
	}
	return nil
}

// Get returns a workspace by ID.
func (ws *Workspaces) Get(id string) (*Workspace, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	w, ok := ws.workspaces[id]
	return w, ok
}

// Active returns the currently active workspace.
func (ws *Workspaces) Active() *Workspace {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	if ws.active == "" {
		return nil
	}
	return ws.workspaces[ws.active]
}

// SetActive sets the active workspace by ID.
func (ws *Workspaces) SetActive(id string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if _, ok := ws.workspaces[id]; !ok {
		return fmt.Errorf("workspace %q not found", id)
	}
	ws.active = id
	return nil
}

// All returns all workspaces.
func (ws *Workspaces) All() []*Workspace {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	result := make([]*Workspace, 0, len(ws.workspaces))
	for _, w := range ws.workspaces {
		result = append(result, w)
	}
	return result
}

// Count returns the number of managed workspaces.
func (ws *Workspaces) Count() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return len(ws.workspaces)
}

// OnEvent registers an event listener.
func (ws *Workspaces) OnEvent(fn func(WorkspaceEvent)) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.listeners = append(ws.listeners, fn)
}

func (ws *Workspaces) emit(event WorkspaceEvent) {
	for _, fn := range ws.listeners {
		go fn(event)
	}
}

// FindByURI finds a workspace by its URI.
func (ws *Workspaces) FindByURI(uri string) *Workspace {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	for _, w := range ws.workspaces {
		if w.URI == uri {
			return w
		}
	}
	return nil
}

// Open transitions a workspace to the ready state.
func (w *Workspace) Open() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.State = WorkspaceStateReady
	w.OpenedAt = time.Now()
	w.Events = append(w.Events, WorkspaceEvent{
		Kind:      EventWorkspaceOpened,
		URI:       w.URI,
		Timestamp: time.Now(),
	})
}

// Close transitions a workspace to uninitialized state.
func (w *Workspace) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.State = WorkspaceStateUninitialized
	w.Events = append(w.Events, WorkspaceEvent{
		Kind:      EventWorkspaceClosed,
		URI:       w.URI,
		Timestamp: time.Now(),
	})
}

// Path returns a path relative to the workspace root.
func (w *Workspace) Path(d ...string) string {
	p := w.URI
	p = strings.TrimPrefix(p, "file://")
	p = strings.TrimPrefix(p, "file:")
	p = filepath.Clean(p)
	return filepath.Join(p, filepath.Join(d...))
}

// Index returns the path to the workspace index file.
func (w *Workspace) Index() string {
	return w.Path(w.Settings.IndexName)
}

// NotesPath returns the path to the notes directory.
func (w *Workspace) NotesPath() string {
	return w.Path(w.Settings.NotesDir)
}

// TemplatesPath returns the path to the templates directory.
func (w *Workspace) TemplatesPath() string {
	return w.Path(w.Settings.TemplatesDir)
}

// JournalPath returns the path to the journal directory.
func (w *Workspace) JournalPath() string {
	return w.Path(w.Settings.JournalDir)
}

// StorePath returns the internal store path.
func (w *Workspace) StorePath() string {
	return w.Path(w.Settings.StoreDir)
}

// SettingsPath returns the settings file path.
func (w *Workspace) SettingsPath() string {
	return w.Path(w.Settings.StoreDir, "settings.json")
}

// Initialize creates the workspace directory structure.
func (w *Workspace) Initialize() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.State = WorkspaceStateInitializing

	dirs := []string{
		w.Settings.NotesDir,
		w.Settings.TemplatesDir,
		w.Settings.SnippetsDir,
		w.Settings.JournalDir,
		w.Settings.AttachmentsDir,
		w.Settings.StoreDir,
	}

	root := w.URI
	for _, dir := range dirs {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			w.State = WorkspaceStateError
			return fmt.Errorf("failed to create %s: %w", path, err)
		}
	}

	// Write settings
	settingsPath := filepath.Join(root, w.Settings.StoreDir, "settings.json")
	data, err := json.MarshalIndent(w.Settings, "", "  ")
	if err != nil {
		w.State = WorkspaceStateError
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		w.State = WorkspaceStateError
		return fmt.Errorf("failed to write settings: %w", err)
	}

	// Create index if it doesn't exist
	indexPath := filepath.Join(root, w.Settings.IndexName)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		indexContent := fmt.Sprintf("# %s\n\nWorkspace index.\n", w.Name)
		if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
			w.State = WorkspaceStateError
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	w.State = WorkspaceStateReady
	return nil
}

// LoadSettings reads settings from the workspace store directory.
func (w *Workspace) LoadSettings() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	settingsPath := filepath.Join(w.URI, w.Settings.StoreDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // use defaults
		}
		return err
	}

	var settings WorkspaceSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	w.Settings = settings
	return nil
}

// SaveSettings writes settings to the workspace store directory.
func (w *Workspace) SaveSettings() error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	dir := filepath.Join(w.URI, w.Settings.StoreDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	settingsPath := filepath.Join(dir, "settings.json")
	data, err := json.MarshalIndent(w.Settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0644)
}

// RecordEvent records a workspace event.
func (w *Workspace) RecordEvent(kind WorkspaceEventKind, uri string, data interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	event := WorkspaceEvent{
		Kind:      kind,
		URI:       uri,
		Timestamp: time.Now(),
		Data:      data,
	}
	w.Events = append(w.Events, event)
	// Keep only last 100 events
	if len(w.Events) > 100 {
		w.Events = w.Events[len(w.Events)-100:]
	}
}

// IsExcluded checks if a path should be excluded based on workspace settings.
func (w *Workspace) IsExcluded(path string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	base := filepath.Base(path)
	for _, glob := range w.Settings.ExcludeGlobs {
		if matched, _ := filepath.Match(glob, base); matched {
			return true
		}
	}
	return false
}

// Legacy compatibility alias.
type LegacyWorkspace = store.Store[Name, store.Store[Identifier, interface{}]]
