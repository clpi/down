package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Profile represents a user profile with preferences and settings.
type Profile struct {
	mu          sync.RWMutex
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Email       string            `json:"email,omitempty"`
	Avatar      string            `json:"avatar,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Preferences Preferences       `json:"preferences"`
	AISettings  AISettings        `json:"ai_settings"`
	Workspaces  []WorkspaceRef    `json:"workspaces"`
	RecentFiles []RecentFile      `json:"recent_files,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	CustomMeta  map[string]string `json:"custom_meta,omitempty"`
	storePath   string
}

// Preferences holds user-configurable preferences.
type Preferences struct {
	Theme           string `json:"theme"`
	DefaultTemplate string `json:"default_template"`
	DateFormat      string `json:"date_format"`
	TimeFormat      string `json:"time_format"`
	Language        string `json:"language"`
	AutoSave        bool   `json:"auto_save"`
	AutoIndex       bool   `json:"auto_index"`
	ShowInlayHints  bool   `json:"show_inlay_hints"`
	ShowCodeLens    bool   `json:"show_code_lens"`
	ShowDiagnostics bool   `json:"show_diagnostics"`
	MaxCompletions  int    `json:"max_completions"`
	TabSize         int    `json:"tab_size"`
	WordWrap        bool   `json:"word_wrap"`
	SpellCheck      bool   `json:"spell_check"`
}

// AISettings configures AI behavior per user.
type AISettings struct {
	Enabled       bool    `json:"enabled"`
	Provider      string  `json:"provider"`
	Model         string  `json:"model,omitempty"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	Completions   bool    `json:"completions"`
	Suggestions   bool    `json:"suggestions"`
	SystemPrompt  string  `json:"system_prompt,omitempty"`
	ContextWindow int     `json:"context_window"`
}

// WorkspaceRef is a reference to a workspace in the user profile.
type WorkspaceRef struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	URI      string    `json:"uri"`
	LastOpen time.Time `json:"last_open"`
	Pinned   bool      `json:"pinned"`
}

// RecentFile tracks recently accessed files.
type RecentFile struct {
	URI        string    `json:"uri"`
	Name       string    `json:"name"`
	Workspace  string    `json:"workspace"`
	AccessedAt time.Time `json:"accessed_at"`
}

// DefaultPreferences returns sensible default preferences.
func DefaultPreferences() Preferences {
	return Preferences{
		Theme:           "default",
		DefaultTemplate: "note",
		DateFormat:       "2006-01-02",
		TimeFormat:       "15:04:05",
		Language:         "en",
		AutoSave:         true,
		AutoIndex:        true,
		ShowInlayHints:   true,
		ShowCodeLens:     true,
		ShowDiagnostics:  true,
		MaxCompletions:   20,
		TabSize:          2,
		WordWrap:         true,
		SpellCheck:       false,
	}
}

// DefaultAISettings returns sensible AI defaults.
func DefaultAISettings() AISettings {
	return AISettings{
		Enabled:       true,
		Provider:      "auto",
		Temperature:   0.7,
		MaxTokens:     1024,
		Completions:   true,
		Suggestions:   true,
		ContextWindow: 20,
	}
}

// NewProfile creates a new user profile with defaults.
func NewProfile(name string) *Profile {
	configDir, _ := os.UserConfigDir()
	storePath := filepath.Join(configDir, "down", "profile.json")

	return &Profile{
		ID:          generateID(name),
		Name:        name,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Preferences: DefaultPreferences(),
		AISettings:  DefaultAISettings(),
		Workspaces:  make([]WorkspaceRef, 0),
		RecentFiles: make([]RecentFile, 0),
		Tags:        make(map[string]string),
		CustomMeta:  make(map[string]string),
		storePath:   storePath,
	}
}

// LoadProfile loads a profile from the default location.
func LoadProfile() (*Profile, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine config directory: %w", err)
	}
	storePath := filepath.Join(configDir, "down", "profile.json")
	return LoadProfileFrom(storePath)
}

// LoadProfileFrom loads a profile from a specific path.
func LoadProfileFrom(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return a default profile
			p := NewProfile("default")
			p.storePath = path
			return p, nil
		}
		return nil, err
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("invalid profile: %w", err)
	}
	profile.storePath = path
	return &profile, nil
}

// Save persists the profile to disk.
func (p *Profile) Save() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.UpdatedAt = time.Now()

	dir := filepath.Dir(p.storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.storePath, data, 0644)
}

// SetPreference sets a single preference value.
func (p *Profile) SetPreference(key string, value interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch key {
	case "theme":
		if v, ok := value.(string); ok {
			p.Preferences.Theme = v
		}
	case "date_format":
		if v, ok := value.(string); ok {
			p.Preferences.DateFormat = v
		}
	case "time_format":
		if v, ok := value.(string); ok {
			p.Preferences.TimeFormat = v
		}
	case "language":
		if v, ok := value.(string); ok {
			p.Preferences.Language = v
		}
	case "auto_save":
		if v, ok := value.(bool); ok {
			p.Preferences.AutoSave = v
		}
	case "auto_index":
		if v, ok := value.(bool); ok {
			p.Preferences.AutoIndex = v
		}
	case "show_inlay_hints":
		if v, ok := value.(bool); ok {
			p.Preferences.ShowInlayHints = v
		}
	case "show_code_lens":
		if v, ok := value.(bool); ok {
			p.Preferences.ShowCodeLens = v
		}
	case "max_completions":
		if v, ok := value.(int); ok {
			p.Preferences.MaxCompletions = v
		}
	case "tab_size":
		if v, ok := value.(int); ok {
			p.Preferences.TabSize = v
		}
	case "word_wrap":
		if v, ok := value.(bool); ok {
			p.Preferences.WordWrap = v
		}
	case "spell_check":
		if v, ok := value.(bool); ok {
			p.Preferences.SpellCheck = v
		}
	default:
		return fmt.Errorf("unknown preference: %s", key)
	}

	p.UpdatedAt = time.Now()
	return nil
}

// AddWorkspace registers a workspace in the profile.
func (p *Profile) AddWorkspace(id, name, uri string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if already exists
	for i, ws := range p.Workspaces {
		if ws.ID == id || ws.URI == uri {
			p.Workspaces[i].LastOpen = time.Now()
			p.UpdatedAt = time.Now()
			return
		}
	}

	p.Workspaces = append(p.Workspaces, WorkspaceRef{
		ID:       id,
		Name:     name,
		URI:      uri,
		LastOpen: time.Now(),
	})
	p.UpdatedAt = time.Now()
}

// RemoveWorkspace removes a workspace from the profile.
func (p *Profile) RemoveWorkspace(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	filtered := p.Workspaces[:0]
	for _, ws := range p.Workspaces {
		if ws.ID != id {
			filtered = append(filtered, ws)
		}
	}
	p.Workspaces = filtered
	p.UpdatedAt = time.Now()
}

// RecordFileAccess records a file access for recent files tracking.
func (p *Profile) RecordFileAccess(uri, name, workspace string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove if already in list
	filtered := p.RecentFiles[:0]
	for _, rf := range p.RecentFiles {
		if rf.URI != uri {
			filtered = append(filtered, rf)
		}
	}

	// Add to front
	filtered = append([]RecentFile{{
		URI:        uri,
		Name:       name,
		Workspace:  workspace,
		AccessedAt: time.Now(),
	}}, filtered...)

	// Keep only last 50
	if len(filtered) > 50 {
		filtered = filtered[:50]
	}
	p.RecentFiles = filtered
	p.UpdatedAt = time.Now()
}

func (p *Profile) GetRecentFiles(limit int) []RecentFile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if limit <= 0 || limit > len(p.RecentFiles) {
		limit = len(p.RecentFiles)
	}
	result := make([]RecentFile, limit)
	copy(result, p.RecentFiles[:limit])
	return result
}

// Summary returns a text summary of the profile.
func (p *Profile) Summary() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var sb fmt.Stringer = &profileSummaryBuilder{p: p}
	return sb.String()
}

type profileSummaryBuilder struct {
	p *Profile
}

func (psb *profileSummaryBuilder) String() string {
	p := psb.p
	s := fmt.Sprintf("## User Profile: %s\n\n", p.Name)
	s += fmt.Sprintf("- ID: %s\n", p.ID)
	if p.Email != "" {
		s += fmt.Sprintf("- Email: %s\n", p.Email)
	}
	s += fmt.Sprintf("- Created: %s\n", p.CreatedAt.Format("2006-01-02"))
	s += fmt.Sprintf("- Workspaces: %d\n", len(p.Workspaces))
	s += fmt.Sprintf("- Recent files: %d\n", len(p.RecentFiles))
	s += fmt.Sprintf("\n### Preferences\n")
	s += fmt.Sprintf("- Theme: %s\n", p.Preferences.Theme)
	s += fmt.Sprintf("- Language: %s\n", p.Preferences.Language)
	s += fmt.Sprintf("- AI enabled: %v\n", p.AISettings.Enabled)
	s += fmt.Sprintf("- AI provider: %s\n", p.AISettings.Provider)
	return s
}

func generateID(name string) string {
	// Simple deterministic ID from name + timestamp
	ts := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d", sanitizeID(name), ts%1000000)
}

func sanitizeID(s string) string {
	var result []byte
	for _, ch := range []byte(s) {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			result = append(result, ch)
		} else if ch >= 'A' && ch <= 'Z' {
			result = append(result, ch+32)
		} else if ch == ' ' || ch == '_' {
			result = append(result, '-')
		}
	}
	if len(result) == 0 {
		return "user"
	}
	return string(result)
}
