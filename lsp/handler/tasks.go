package handler

import (
	"regexp"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Task represents a markdown task item (e.g. - [ ] or - [x])
type Task struct {
	URI       string `json:"uri"`
	Title     string `json:"title"`
	Line      int    `json:"line"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// TasksResult contains the list of all found tasks
type TasksResult struct {
	Tasks []Task `json:"tasks"`
	Count int    `json:"count"`
}

var reTask = regexp.MustCompile(`^\s*[-*+]\s+\[([ xX])\]\s+(.*)`)

// ComputeTasks scans all known documents for task items
func (s *State) ComputeTasks() *TasksResult {
	result := &TasksResult{
		Tasks: make([]Task, 0),
	}

	for uri, text := range s.Documents {
		title := getDocTitle(s.Documents, uri)
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			m := reTask.FindStringSubmatch(line)
			if m != nil {
				completed := m[1] != " "
				result.Tasks = append(result.Tasks, Task{
					URI:       uri,
					Title:     title,
					Line:      i,
					Text:      strings.TrimSpace(m[2]),
					Completed: completed,
				})
			}
		}
	}
	result.Count = len(result.Tasks)
	return result
}

// TaskToggleEdit builds a workspace edit that toggles a task checkbox on the given line.
func (s *State) TaskToggleEdit(uri string, line int) *protocol.WorkspaceEdit {
	text, ok := s.Documents[uri]
	if !ok {
		return nil
	}
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return nil
	}

	current := lines[line]
	m := reTask.FindStringSubmatchIndex(current)
	if m == nil {
		return nil
	}

	checkStart := strings.Index(current[m[0]:m[1]], "[") + m[0]
	checkEnd := checkStart + 3
	if checkEnd > len(current) {
		return nil
	}
	checkbox := current[checkStart:checkEnd]
	var newCheckbox string
	switch checkbox {
	case "[ ]":
		newCheckbox = "[x]"
	case "[x]", "[X]":
		newCheckbox = "[ ]"
	default:
		return nil
	}

	return &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentUri][]protocol.TextEdit{
			protocol.DocumentUri(uri): {
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(checkStart)},
						End:   protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(checkEnd)},
					},
					NewText: newCheckbox,
				},
			},
		},
	}
}
