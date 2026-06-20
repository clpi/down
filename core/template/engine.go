package template

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Template represents a template file with metadata.
type Template struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Type        string            `json:"type"`
	Category    string            `json:"category"`
	Description string            `json:"description"`
	Variables   []string          `json:"variables"`
	Frontmatter map[string]string `json:"frontmatter"`
	Content     string            `json:"content"`
	Source      string            `json:"source"`
}

// Engine manages template loading, variable expansion, and application.
type Engine struct {
	Dirs    []string
	Vars    VariableRegistry
	Profile ProfileProvider
}

// ProfileProvider provides user profile data for template expansion.
type ProfileProvider interface {
	Get(key string) string
}

// VariableRegistry maps variable names to resolver functions.
type VariableRegistry map[string]func() string

// NewEngine creates a template engine scanning the given directories.
func NewEngine(dirs ...string) *Engine {
	e := &Engine{
		Dirs: dirs,
		Vars: make(VariableRegistry),
	}
	e.registerDefaults()
	return e
}

func (e *Engine) registerDefaults() {
	now := time.Now()
	e.Vars["date"] = func() string { return now.Format("2006-01-02") }
	e.Vars["time"] = func() string { return now.Format("15:04:05") }
	e.Vars["datetime"] = func() string { return now.Format("2006-01-02 15:04:05") }
	e.Vars["year"] = func() string { return fmt.Sprintf("%d", now.Year()) }
	e.Vars["month"] = func() string { return fmt.Sprintf("%02d", now.Month()) }
	e.Vars["day"] = func() string { return fmt.Sprintf("%02d", now.Day()) }
	e.Vars["weekday"] = func() string { return now.Weekday().String() }
	e.Vars["iso_year"] = func() string {
		y, _ := now.ISOWeek()
		return fmt.Sprintf("%d", y)
	}
	e.Vars["iso_week"] = func() string {
		_, w := now.ISOWeek()
		return fmt.Sprintf("%02d", w)
	}
	e.Vars["timestamp"] = func() string { return fmt.Sprintf("%d", now.Unix()) }
}

// Register adds a custom variable resolver.
func (e *Engine) Register(name string, resolver func() string) {
	e.Vars[name] = resolver
}

// Load all templates from configured directories.
func (e *Engine) Load() []Template {
	var templates []Template
	seen := make(map[string]bool)

	for _, dir := range e.Dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".md")
			if seen[name] {
				continue
			}
			seen[name] = true

			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			tmpl := Template{
				Name:   name,
				Path:   path,
				Source: dir,
			}
			tmpl.parse(string(data))
			templates = append(templates, tmpl)
		}
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	return templates
}

// Find locates a template by name.
func (e *Engine) Find(name string) *Template {
	for _, dir := range e.Dirs {
		path := filepath.Join(dir, name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		tmpl := &Template{
			Name:   name,
			Path:   path,
			Source: dir,
		}
		tmpl.parse(string(data))
		return tmpl
	}
	return nil
}

// parse extracts metadata from template frontmatter and body.
func (t *Template) parse(raw string) {
	t.Content = raw

	if !strings.HasPrefix(raw, "---\n") {
		return
	}

	end := strings.Index(raw[4:], "\n---\n")
	if end <= 0 {
		return
	}

	fm := raw[4 : 4+end]
	t.Frontmatter = make(map[string]string)
	for _, line := range strings.Split(fm, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		t.Frontmatter[key] = val

		switch key {
		case "type":
			t.Type = val
		case "category":
			t.Category = val
		case "description":
			t.Description = val
		case "variables":
			for _, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				if v != "" {
					t.Variables = append(t.Variables, v)
				}
			}
		}
	}

	t.Content = raw[4+end+5:]
}

// Expand substitutes template variables in the content.
func (e *Engine) Expand(tmpl *Template, extra map[string]string) string {
	content := tmpl.Content

	for name, fn := range e.Vars {
		placeholder := "{{" + name + "}}"
		if strings.Contains(content, placeholder) {
			content = strings.ReplaceAll(content, placeholder, fn())
		}
	}

	for k, v := range extra {
		placeholder := "{{" + k + "}}"
		content = strings.ReplaceAll(content, placeholder, v)
	}

	return content
}

// Apply loads a template by name, expands variables, and returns result.
func (e *Engine) Apply(name string, extra map[string]string) (string, error) {
	tmpl := e.Find(name)
	if tmpl == nil {
		tmpl = builtinTemplate(name)
	}
	if tmpl == nil {
		return "", fmt.Errorf("template %q not found", name)
	}
	return e.Expand(tmpl, extra), nil
}

// ApplyToFile applies a template and writes to the given path.
func (e *Engine) ApplyToFile(name, dest string, extra map[string]string) error {
	content, err := e.Apply(name, extra)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(dest, []byte(content), 0644)
}

// Create writes a new template to the first writable directory.
func (e *Engine) Create(name string, content string, meta map[string]string) (string, error) {
	if len(e.Dirs) == 0 {
		return "", fmt.Errorf("no template directories configured")
	}

	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString(fmt.Sprintf("type: %s\n", orDefault(meta["type"], "note")))
	if cat := meta["category"]; cat != "" {
		fm.WriteString(fmt.Sprintf("category: %s\n", cat))
	}
	if desc := meta["description"]; desc != "" {
		fm.WriteString(fmt.Sprintf("description: %s\n", desc))
	}
	if vars := meta["variables"]; vars != "" {
		fm.WriteString(fmt.Sprintf("variables: %s\n", vars))
	}
	fm.WriteString("---\n\n")
	fm.WriteString(content)

	dir := e.Dirs[0]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+".md")
	return path, os.WriteFile(path, []byte(fm.String()), 0644)
}

// Delete removes a template by name from all directories.
func (e *Engine) Delete(name string) error {
	var deleted bool
	for _, dir := range e.Dirs {
		path := filepath.Join(dir, name+".md")
		if err := os.Remove(path); err == nil {
			deleted = true
		}
	}
	if !deleted {
		return fmt.Errorf("template %q not found", name)
	}
	return nil
}

// Validate checks a template for syntax errors and required fields.
func (e *Engine) Validate(tmpl *Template) []string {
	var issues []string

	if tmpl.Content == "" {
		issues = append(issues, "template content is empty")
	}

	if tmpl.Type == "" {
		issues = append(issues, "template has no type (add type: note|daily|meeting|project|etc to frontmatter)")
	}

	for _, line := range strings.Split(tmpl.Content, "\n") {
		opens := strings.Count(line, "{{")
		closes := strings.Count(line, "}}")
		if opens != closes {
			issues = append(issues, fmt.Sprintf("unmatched {{ }} on line: %s", line))
		}
	}

	return issues
}

// Categories returns all distinct categories found in templates.
func (e *Engine) Categories() []string {
	seen := make(map[string]bool)
	for _, t := range e.Load() {
		if t.Category != "" {
			seen[t.Category] = true
		}
	}
	var cats []string
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

// Types returns all distinct types found in templates.
func (e *Engine) Types() []string {
	seen := make(map[string]bool)
	for _, t := range e.Load() {
		if t.Type != "" {
			seen[t.Type] = true
		}
	}
	var types []string
	for t := range seen {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func orDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

// Builtin returns a built-in template by name, or nil.
func Builtin(name string) *Template {
	return builtinTemplate(name)
}

// ExpandString expands variables in a raw string using the given time.
func ExpandString(content string, t time.Time) string {
	c := content
	c = strings.ReplaceAll(c, "{{date}}", t.Format("2006-01-02"))
	c = strings.ReplaceAll(c, "{{time}}", t.Format("15:04:05"))
	c = strings.ReplaceAll(c, "{{datetime}}", t.Format("2006-01-02 15:04:05"))
	c = strings.ReplaceAll(c, "{{year}}", fmt.Sprintf("%d", t.Year()))
	c = strings.ReplaceAll(c, "{{month}}", fmt.Sprintf("%02d", t.Month()))
	c = strings.ReplaceAll(c, "{{day}}", fmt.Sprintf("%02d", t.Day()))
	c = strings.ReplaceAll(c, "{{weekday}}", t.Weekday().String())
	return c
}

func builtinTemplate(name string) *Template {
	builtins := map[string]string{
		"meeting": `# Meeting: {{title}}

**Date:** {{date}}
**Time:** {{time}}
**Attendees:** 

## Agenda

- 

## Notes

## Action Items

- [ ] 
`,
		"daily": `# {{date}}

## Morning

## Work Log

- 

## Evening

## Gratitude

- 
`,
		"project": `# Project: {{title}}

**Status:** in-progress
**Start:** {{date}}
**Target:** 

## Goals

- 

## Timeline

| Phase | Status | Date |
|-------|--------|------|
| Planning | done | |
| Execution | in-progress | |
| Review | pending | |

## Notes
`,
		"note": `# {{title}}

**Date:** {{date}}
**Tags:** 

`,
		"research": `# Research: {{title}}

**Date:** {{date}}
**Source:** 

## Key Findings

- 

## Notes

## References

- 
`,
		"weekly": `# Week {{iso_year}}-W{{iso_week}}

**{{date}}**

## Highlights

- 

## Monday
## Tuesday
## Wednesday
## Thursday
## Friday

## Next Week
`,
		"monthly": `# {{month}} {{year}} Review

## Highlights

- 

## By Week

### Week 1
### Week 2
### Week 3
### Week 4

## Stats

## Next Month
`,
	}

	content, ok := builtins[name]
	if !ok {
		return nil
	}
	return &Template{
		Name:        name,
		Type:        name,
		Description: fmt.Sprintf("Built-in %s template", name),
		Content:     content,
		Source:      "builtin",
	}
}
