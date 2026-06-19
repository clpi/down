package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clpi/down/lsp/knowledge"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func dataDir() string {
	d := os.Getenv("XDG_DATA_HOME")
	if d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(d, "down")
}

func workspaceDir() string {
	cwd, _ := os.Getwd()
	for dir := cwd; dir != "" && dir != "/" && filepath.Dir(dir) != dir; {
		dd := filepath.Join(dir, ".down")
		if info, err := os.Stat(dd); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir { break }
		dir = parent
	}
	return cwd
}

func getArgs(req mcp.CallToolRequest) map[string]interface{} {
	if args, ok := req.Params.Arguments.(map[string]interface{}); ok {
		return args
	}
	return map[string]interface{}{}
}

func strArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func boolArg(args map[string]interface{}, key string) (bool, bool) {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok { return b, true }
	}
	return false, false
}

func loadKnowledge() *knowledge.Graph {
	path := filepath.Join(dataDir(), "knowledge.json")
	g := knowledge.NewGraph(path)
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, g)
	}
	return g
}

func saveKnowledge(g *knowledge.Graph) {
	path := filepath.Join(dataDir(), "knowledge.json")
	os.MkdirAll(dataDir(), 0755)
	data, _ := json.MarshalIndent(g, "", "  ")
	os.WriteFile(path, data, 0644)
}

func memoryDir() string {
	return filepath.Join(dataDir(), "memory")
}

type MemoryEntry struct {
	Key     string   `json:"key"`
	Value   string   `json:"value"`
	Tags    []string `json:"tags,omitempty"`
	Created string   `json:"created_at,omitempty"`
}

func loadMemory(key string) (*MemoryEntry, error) {
	data, err := os.ReadFile(filepath.Join(memoryDir(), key+".json"))
	if err != nil {
		return nil, err
	}
	var e MemoryEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func listMemory() []MemoryEntry {
	entries, _ := os.ReadDir(memoryDir())
	var mems []MemoryEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" { continue }
		entry, err := loadMemory(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil { continue }
		mems = append(mems, *entry)
	}
	sort.Slice(mems, func(i, j int) bool { return mems[i].Created > mems[j].Created })
	return mems
}

func searchMemory(query string) []MemoryEntry {
	mems := listMemory()
	lower := strings.ToLower(query)
	var results []MemoryEntry
	for _, m := range mems {
		if strings.Contains(strings.ToLower(m.Key), lower) || strings.Contains(strings.ToLower(m.Value), lower) {
			results = append(results, m)
			continue
		}
		for _, t := range m.Tags {
			if strings.Contains(strings.ToLower(t), lower) {
				results = append(results, m)
				break
			}
		}
	}
	return results
}

func findMarkdownFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if info.IsDir() {
			n := info.Name()
			if strings.HasPrefix(n, ".") || n == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func tasksInDir(dir string) []map[string]interface{} {
	var tasks []map[string]interface{}
	for _, file := range findMarkdownFiles(dir) {
		data, err := os.ReadFile(file)
		if err != nil { continue }
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "- [") { continue }
			done := strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]")
			content := strings.TrimPrefix(trimmed, "- [ ] ")
			content = strings.TrimPrefix(content, "- [x] ")
			content = strings.TrimPrefix(content, "- [X] ")
			tasks = append(tasks, map[string]interface{}{
				"file": file, "line": i + 1, "done": done, "content": content,
			})
		}
	}
	return tasks
}

func Run() {
	s := server.NewMCPServer(
		"down.nvim",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// === Workspace tools ===

	s.AddTool(mcp.NewTool("list_workspaces",
		mcp.WithDescription("List all configured down workspaces"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wd := workspaceDir()
		return mcp.NewToolResultText(fmt.Sprintf("Current workspace: %s\nRoot: %s", filepath.Base(wd), wd)), nil
	})

	s.AddTool(mcp.NewTool("search_notes",
		mcp.WithDescription("Search across all markdown notes in the workspace"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query string")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		query := strArg(args, "query")
		wd := workspaceDir()
		files := findMarkdownFiles(wd)
		var results []string
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil { continue }
			if strings.Contains(strings.ToLower(string(data)), strings.ToLower(query)) {
				results = append(results, fmt.Sprintf("### %s\nFound match", file))
			}
		}
		if len(results) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No results for: %s", query)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Found in %d files:\n%s", len(results), strings.Join(results, "\n"))), nil
	})

	s.AddTool(mcp.NewTool("read_note",
		mcp.WithDescription("Read the full content of a note by path"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative or absolute path to the note")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		path := strArg(args, "path")
		if !filepath.IsAbs(path) {
			path = filepath.Join(workspaceDir(), path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("# %s\n\n%s", filepath.Base(path), string(data))), nil
	})

	// === Knowledge graph tools ===

	s.AddTool(mcp.NewTool("knowledge_graph_search",
		mcp.WithDescription("Search the knowledge graph for entities and relations"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Entity name or keyword")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		query := strArg(args, "query")
		g := loadKnowledge()
		results := g.Search(query)
		if len(results) == 0 {
			return mcp.NewToolResultText("No knowledge graph results"), nil
		}
		var lines []string
		for _, e := range results {
			lines = append(lines, fmt.Sprintf("- **%s** (%s) — %d mentions", e.Name, e.Kind, e.Mentions))
		}
		return mcp.NewToolResultText(fmt.Sprintf("Results for \"%s\":\n%s", query, strings.Join(lines, "\n"))), nil
	})

	s.AddTool(mcp.NewTool("knowledge_graph_stats",
		mcp.WithDescription("Get knowledge graph statistics"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		g := loadKnowledge()
		return mcp.NewToolResultText(g.Summary()), nil
	})

	// === Memory tools ===

	s.AddTool(mcp.NewTool("add_memory",
		mcp.WithDescription("Store a persistent memory entry for AI context"),
		mcp.WithString("key", mcp.Required(), mcp.Description("Memory key")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Memory content")),
		mcp.WithArray("tags", mcp.Description("Optional tags")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		key := strArg(args, "key")
		value := strArg(args, "value")
		var tags []string
		if t, ok := args["tags"].([]interface{}); ok {
			for _, tag := range t { tags = append(tags, fmt.Sprint(tag)) }
		}
		e := MemoryEntry{Key: key, Value: value, Tags: tags, Created: time.Now().Format("2006-01-02 15:04")}
		os.MkdirAll(memoryDir(), 0755)
		data, _ := json.MarshalIndent(e, "", "  ")
		os.WriteFile(filepath.Join(memoryDir(), key+".json"), data, 0644)
		return mcp.NewToolResultText(fmt.Sprintf("Memory saved: %s", key)), nil
	})

	s.AddTool(mcp.NewTool("search_memory",
		mcp.WithDescription("Search persistent memory entries"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		query := strArg(args, "query")
		results := searchMemory(query)
		if len(results) == 0 {
			return mcp.NewToolResultText("No memory entries found"), nil
		}
		var lines []string
		for _, e := range results {
			preview := e.Value
			if len(preview) > 300 { preview = preview[:300] + "..." }
			lines = append(lines, fmt.Sprintf("## %s\n%s\n", e.Key, preview))
		}
		return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
	})

	s.AddTool(mcp.NewTool("list_memory",
		mcp.WithDescription("List all persistent memory entries"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mems := listMemory()
		if len(mems) == 0 {
			return mcp.NewToolResultText("No memory entries"), nil
		}
		var lines []string
		for _, e := range mems {
			tagStr := ""
			if len(e.Tags) > 0 { tagStr = fmt.Sprintf(" [%s]", strings.Join(e.Tags, ", ")) }
			lines = append(lines, fmt.Sprintf("- %s%s — %s", e.Key, tagStr, e.Created))
		}
		return mcp.NewToolResultText(fmt.Sprintf("Memory entries (%d):\n%s", len(mems), strings.Join(lines, "\n"))), nil
	})

	// === Task tools ===

	s.AddTool(mcp.NewTool("list_tasks",
		mcp.WithDescription("List all tasks across workspace notes"),
		mcp.WithBoolean("done", mcp.Description("Filter by completion status")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		wd := workspaceDir()
		tasks := tasksInDir(wd)
		if len(tasks) == 0 {
			return mcp.NewToolResultText("No tasks found"), nil
		}
		filterDone, hasFilter := boolArg(args, "done")
		var lines []string
		count := 0
		for _, t := range tasks {
			if hasFilter && t["done"].(bool) != filterDone { continue }
			marker := "[ ]"
			if t["done"].(bool) { marker = "[x]" }
			lines = append(lines, fmt.Sprintf("- %s %s _(%s:%v)_", marker, t["content"], t["file"], t["line"]))
			count++
		}
		return mcp.NewToolResultText(fmt.Sprintf("Tasks (%d):\n%s", count, strings.Join(lines, "\n"))), nil
	})

	// === Note creation ===

	s.AddTool(mcp.NewTool("create_note",
		mcp.WithDescription("Create a new markdown note in the workspace"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Note title")),
		mcp.WithString("content", mcp.Description("Initial content")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		title := strArg(args, "title")
		content := strArg(args, "content")
		wd := workspaceDir()
		filename := strings.ToLower(strings.ReplaceAll(title, " ", "-")) + ".md"
		path := filepath.Join(wd, filename)
		header := fmt.Sprintf("# %s\n\nCreated: %s\n\n", title, time.Now().Format("2006-01-02 15:04"))
		os.WriteFile(path, []byte(header+content), 0644)
		return mcp.NewToolResultText(fmt.Sprintf("Created note: %s", path)), nil
	})

	// === Compact ===

	s.AddTool(mcp.NewTool("compact_workspace",
		mcp.WithDescription("Generate compact markdown of workspace notes for AI"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wd := workspaceDir()
		files := findMarkdownFiles(wd)
		var lines []string
		for _, f := range files {
			rel, _ := filepath.Rel(wd, f)
			data, _ := os.ReadFile(f)
			content := string(data)
			if len(content) > 2000 { content = content[:2000] + "\n...(truncated)" }
			lines = append(lines, fmt.Sprintf("### %s\n```markdown\n%s\n```", rel, content))
		}
		return mcp.NewToolResultText(strings.Join(lines, "\n\n")), nil
	})

	log.Println("MCP server starting on stdio...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}

var Mcp = cobra.Command{
	Use:    "mcp",
	Short:  "Start MCP server",
	Long:   "Start the Model Context Protocol server with 10+ tools for workspace, knowledge graph, memory, tasks, and notes.",
	Aliases: []string{"m"},
	Run: func(cmd *cobra.Command, args []string) {
		Run()
	},
}

func init() {
	Mcp.Flags().String("data-dir", "", "Override data directory")
}
