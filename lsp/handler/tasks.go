package handler

import (
	"regexp"
	"strings"
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
