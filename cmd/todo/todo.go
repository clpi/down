package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	todoDone    bool
	todoGlobal  bool
	todoDue     string
	todoPrio    string
	todoTag     string
)

type Task struct {
	File     string
	Line     int
	Done     bool
	Content  string
	Priority string
	Due      string
	Tags     []string
}

func findTasks(root string, global bool) []Task {
	var tasks []Task
	var roots []string
	if global {
		home, _ := os.UserHomeDir()
		roots = []string{root, filepath.Join(home, ".local", "share", "down", "workspace")}
	} else {
		roots = []string{root}
	}

	seen := map[string]bool{}
	for _, r := range roots {
		if _, err := os.Stat(r); os.IsNotExist(err) { continue }
		filepath.Walk(r, func(path string, info os.FileInfo, err error) error {
			if err != nil { return nil }
			if info.IsDir() {
				n := info.Name()
				if strings.HasPrefix(n, ".") || n == "node_modules" { return filepath.SkipDir }
				return nil
			}
			if !strings.HasSuffix(info.Name(), ".md") { return nil }
			if seen[path] { return nil }
			seen[path] = true

			data, _ := os.ReadFile(path)
			lines := strings.Split(string(data), "\n")
			var currentHeading string
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "#") {
					currentHeading = strings.TrimPrefix(strings.TrimPrefix(trimmed, "# "), "## ")
				}
				if !strings.HasPrefix(trimmed, "- [") { continue }
				done := strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]")
				content := strings.TrimPrefix(trimmed, "- [ ] ")
				content = strings.TrimPrefix(content, "- [x] ")
				content = strings.TrimPrefix(content, "- [X] ")

				// Parse metadata from content
				var priority, due string
				var tags []string
				for _, word := range strings.Fields(content) {
					if strings.HasPrefix(word, "#") { tags = append(tags, word) }
					if strings.HasPrefix(word, "A") && len(word) == 1 { priority = "A" }
					if strings.HasPrefix(word, "B") && len(word) == 1 { priority = "B" }
					if strings.HasPrefix(word, "C") && len(word) == 1 { priority = "C" }
					if strings.HasPrefix(word, "D") && len(word) == 1 { priority = "D" }
					if strings.HasPrefix(word, "E") && len(word) == 1 { priority = "E" }
					if strings.HasPrefix(word, "DEADLINE:") { due = strings.TrimPrefix(word, "DEADLINE:") }
					if strings.HasPrefix(word, "DUE:") { due = strings.TrimPrefix(word, "DUE:") }
				}

				rel, _ := filepath.Rel(root, path)
				tasks = append(tasks, Task{
					File: rel, Line: i + 1, Done: done, Content: content,
					Priority: priority, Due: due, Tags: tags,
				})
				_ = currentHeading
			}
			return nil
		})
	}
	return tasks
}

var Todo = cobra.Command{
	Use:     "todo [filter]",
	Aliases: []string{"tasks", "td"},
	Short:   "Manage todos across workspace or globally",
	Long:    "List, filter, and manage todo items from markdown task lists across the workspace or globally.",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		tasks := findTasks(root, todoGlobal)
		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return
		}

		// Filter
		var filtered []Task
		for _, t := range tasks {
			if todoDone && !t.Done { continue }
			if !todoDone && t.Done { continue }
			if todoPrio != "" && t.Priority != todoPrio { continue }
			if todoTag != "" {
				hasTag := false
				for _, tg := range t.Tags {
					if tg == todoTag { hasTag = true }
				}
				if !hasTag { continue }
			}
			if len(args) > 0 {
				query := strings.ToLower(strings.Join(args, " "))
				if !strings.Contains(strings.ToLower(t.Content), query) { continue }
			}
			filtered = append(filtered, t)
		}

		if len(filtered) == 0 {
			fmt.Println("No matching tasks")
			return
		}

		// Sort: priority first, then alphabetical
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].Priority != filtered[j].Priority {
				return filtered[i].Priority < filtered[j].Priority
			}
			return filtered[i].Content < filtered[j].Content
		})

		// Group by file
		fmt.Printf("Tasks (%d):\n\n", len(filtered))
		currentFile := ""
		for _, t := range filtered {
			if t.File != currentFile {
				fmt.Printf("## %s\n\n", t.File)
				currentFile = t.File
			}
			marker := "[ ]"
			if t.Done { marker = "[x]" }
			prioStr := ""
			if t.Priority != "" { prioStr = fmt.Sprintf(" %s", t.Priority) }
			dueStr := ""
			if t.Due != "" { dueStr = fmt.Sprintf(" DEADLINE:%s", t.Due) }
			tagStr := ""
			if len(t.Tags) > 0 { tagStr = " " + strings.Join(t.Tags, " ") }
			fmt.Printf("- %s%s%s%s :%d\n", marker, prioStr, t.Content, dueStr+tagStr, t.Line)
		}

		// Summary
		doneCount, total := 0, len(tasks)
		for _, t := range tasks { if t.Done { doneCount++ } }
		fmt.Printf("\n---\n%d/%d done\n", doneCount, total)
	},
}

var todoAdd = cobra.Command{
	Use:   "add <task>",
	Short: "Add a todo to the workspace index",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		content := strings.Join(args, " ")
		path := filepath.Join(root, ".down", "TODO.md")
		os.MkdirAll(filepath.Dir(path), 0755)

		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil { fmt.Fprintf(os.Stderr, "Error: %v\n", err); return }
		defer f.Close()

		today := time.Now().Format("2006-01-02")
		fmt.Fprintf(f, "- [ ] %s (added: %s)\n", content, today)
		fmt.Printf("Added: %s\n", content)
	},
}

func init() {
	Todo.Flags().BoolVarP(&todoDone, "done", "d", false, "Show only completed tasks")
	Todo.Flags().BoolVarP(&todoGlobal, "global", "g", false, "Search across all profiles")
	Todo.Flags().StringVarP(&todoPrio, "priority", "p", "", "Filter by priority (A-E)")
	Todo.Flags().StringVarP(&todoTag, "tag", "t", "", "Filter by tag")
	Todo.AddCommand(&todoAdd)
}
