package repomix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	output    string
	format    string
	style     string
	noTree    bool
	noTokens  bool
	noStats   bool
	maxSize   int64
	ignoreFile string
)

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".svg": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".zip": true,
	".tar": true, ".gz": true, ".bz2": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".class": true, ".o": true, ".obj": true, ".mp3": true, ".mp4": true, ".mov": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".db": true, ".wasm": true,
	".lock": true,
}

func estimateTokens(text string) int {
	return (len(text) + 3) / 4
}

func isBinary(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if binaryExts[ext] {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()
	buf := make([]byte, 1024)
	n, _ := f.Read(buf)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

func shouldIgnore(path string, ignores []string) bool {
	base := filepath.Base(path)
	for _, pattern := range ignores {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if strings.Contains(pattern, "/") {
			if matched, _ := filepath.Match(pattern, path); matched {
				return true
			}
		}
	}
	return false
}

type fileEntry struct {
	Path  string
	IsDir bool
}

func defaultIgnores() []string {
	return []string{
		".git", ".svn", "node_modules", ".DS_Store", ".down",
		".idea", ".vscode", "dist", "build", "target", "out", "bin",
		"*.lock", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"*.min.js", "*.min.css", "*.map", "*.log",
	}
}

func loadIgnoreFile(root string) []string {
	var patterns []string
	candidates := []string{
		filepath.Join(root, ".repomixignore"),
		filepath.Join(root, ".down", ".repomixignore"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					patterns = append(patterns, line)
				}
			}
			break
		}
	}
	return patterns
}

func collectFiles(root string) ([]fileEntry, error) {
	ignores := append(defaultIgnores(), loadIgnoreFile(root)...)
	if ignoreFile != "" {
		if data, err := os.ReadFile(ignoreFile); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					ignores = append(ignores, line)
				}
			}
		}
	}

	var entries []fileEntry
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		if shouldIgnore(rel, ignores) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		entries = append(entries, fileEntry{Path: rel, IsDir: info.IsDir()})
		return nil
	})
	return entries, err
}

type fileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Tokens  int    `json:"tokens"`
}

type repomixOutput struct {
	Name        string        `json:"name"`
	Timestamp   string        `json:"timestamp"`
	Description string        `json:"description,omitempty"`
	Tree        []string      `json:"tree,omitempty"`
	Files       []fileContent `json:"files"`
	Stats       stats         `json:"stats"`
}

type stats struct {
	TotalFiles  int `json:"totalFiles"`
	TotalTokens int `json:"totalTokens"`
	TotalChars  int `json:"totalChars"`
}

func generateTree(entries []fileEntry) string {
	var b strings.Builder
	b.WriteString(".\n")
	for _, e := range entries {
		depth := strings.Count(e.Path, string(os.PathSeparator))
		for i := 0; i < depth; i++ {
			b.WriteString("  ")
		}
		name := filepath.Base(e.Path)
		if e.IsDir {
			b.WriteString(name + "/\n")
		} else {
			b.WriteString(name + "\n")
		}
	}
	return b.String()
}

func collectFileContents(root string, entries []fileEntry) ([]fileContent, stats) {
	var files []fileContent
	var st stats
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		full := filepath.Join(root, e.Path)
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.Size() > maxSize {
			continue
		}
		if isBinary(full) {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(data)
		if len(strings.TrimSpace(content)) == 0 {
			continue
		}
		tok := estimateTokens(content)
		st.TotalFiles++
		st.TotalTokens += tok
		st.TotalChars += len(content)
		files = append(files, fileContent{
			Path:    e.Path,
			Content: content,
			Tokens:  tok,
		})
	}
	return files, st
}

func packXML(root string) (string, error) {
	entries, err := collectFiles(root)
	if err != nil {
		return "", err
	}
	files, st := collectFileContents(root, entries)

	var b strings.Builder
	fmt.Fprintf(&b, "<?xml version=\"1.0\" encoding=\"UTF-8\"?\u003e\n")
	fmt.Fprintf(&b, "<repository name=\"%s\"\u003e\n", filepath.Base(root))
	fmt.Fprintf(&b, "  <metadata\u003e\n")
	fmt.Fprintf(&b, "    <source\u003e%s</source\u003e\n", root)
	fmt.Fprintf(&b, "    <generated\u003e%s</generated\u003e\n", time.Now().Format("2006-01-02T15:04:05"))
	if !noStats {
		fmt.Fprintf(&b, "    <files\u003e%d</files\u003e\n", st.TotalFiles)
		fmt.Fprintf(&b, "    <tokens\u003e%d</tokens\u003e\n", st.TotalTokens)
	}
	fmt.Fprintf(&b, "  </metadata\u003e\n")

	if !noTree {
		fmt.Fprintf(&b, "  <structure\u003e\n")
		fmt.Fprintf(&b, "  <![CDATA[\n%s]]>\n", generateTree(entries))
		fmt.Fprintf(&b, "  </structure\u003e\n")
	}

	fmt.Fprintf(&b, "  <files\u003e\n")
	for _, f := range files {
		fmt.Fprintf(&b, "    <file path=\"%s\" tokens=\"%d\"\u003e\n", f.Path, f.Tokens)
		fmt.Fprintf(&b, "<![CDATA[\n%s\n]]>\n", f.Content)
		fmt.Fprintf(&b, "    </file\u003e\n")
	}
	fmt.Fprintf(&b, "  </files\u003e\n")
	fmt.Fprintf(&b, "</repository\u003e\n")
	return b.String(), nil
}

func packMarkdown(root string) (string, error) {
	entries, err := collectFiles(root)
	if err != nil {
		return "", err
	}
	files, st := collectFileContents(root, entries)
	name := filepath.Base(root)

	var b strings.Builder
	fmt.Fprintf(&b, "# Repository: %s\n\n", name)
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	if !noTree {
		fmt.Fprintf(&b, "## Directory Structure\n\n```\n%s```\n\n", generateTree(entries))
	}

	fmt.Fprintf(&b, "## Files\n\n")
	for _, f := range files {
		ext := filepath.Ext(f.Path)
		if ext != "" {
			ext = ext[1:]
		}
		fmt.Fprintf(&b, "### %s\n\n```%s\n%s\n```\n\n", f.Path, ext, f.Content)
	}

	if !noStats {
		fmt.Fprintf(&b, "---\n\n**Files:** %d | **Tokens:** ~%d | **Chars:** %d\n\n", st.TotalFiles, st.TotalTokens, st.TotalChars)
	}
	return b.String(), nil
}

func packPlain(root string) (string, error) {
	entries, err := collectFiles(root)
	if err != nil {
		return "", err
	}
	files, st := collectFileContents(root, entries)

	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\nGenerated: %s\n\n", filepath.Base(root), time.Now().Format("2006-01-02 15:04:05"))
	if !noTree {
		b.WriteString(generateTree(entries))
		b.WriteString("\n")
	}
	for _, f := range files {
		fmt.Fprintf(&b, "--- %s ---\n%s\n\n", f.Path, f.Content)
	}
	if !noStats {
		fmt.Fprintf(&b, "Files: %d | Tokens: ~%d | Chars: %d\n", st.TotalFiles, st.TotalTokens, st.TotalChars)
	}
	return b.String(), nil
}

func packJSON(root string) (string, error) {
	entries, err := collectFiles(root)
	if err != nil {
		return "", err
	}
	files, st := collectFileContents(root, entries)

	var tree []string
	if !noTree {
		for _, e := range entries {
			if e.IsDir {
				tree = append(tree, e.Path+"/")
			} else {
				tree = append(tree, e.Path)
			}
		}
	}

	out := repomixOutput{
		Name:        filepath.Base(root),
		Timestamp:   time.Now().Format("2006-01-02T15:04:05"),
		Description: "Packed by down repomix",
		Tree:        tree,
		Files:       files,
		Stats:       st,
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func packTree(root string) (string, error) {
	entries, err := collectFiles(root)
	if err != nil {
		return "", err
	}
	return generateTree(entries), nil
}

var Repomix = cobra.Command{
	Use:     "repomix [directory]",
	Aliases: []string{"pack", "repo", "rmx"},
	Short:   "Pack a repository into AI-friendly format",
	Long:    "Pack a directory into XML, markdown, JSON, or plain text for AI consumption, respecting .repomixignore and excluding binary files.",
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		var result string
		var err error
		switch style {
		case "markdown", "md":
			result, err = packMarkdown(root)
		case "json":
			result, err = packJSON(root)
		case "plain", "text":
			result, err = packPlain(root)
		case "tree":
			result, err = packTree(root)
		default:
			result, err = packXML(root)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if output != "" {
			if err := os.WriteFile(output, []byte(result), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Wrote %s\n", output)
		} else {
			fmt.Print(result)
		}
	},
}

func init() {
	Repomix.Flags().StringVarP(&output, "output", "o", "", "Write output to file (default: stdout)")
	Repomix.Flags().StringVarP(&format, "format", "f", "xml", "Output format: xml, markdown, json, plain, tree")
	Repomix.Flags().StringVarP(&style, "style", "s", "xml", "Output style: xml, markdown, json, plain, tree")
	Repomix.Flags().BoolVar(&noTree, "no-tree", false, "Omit directory tree")
	Repomix.Flags().BoolVar(&noTokens, "no-tokens", false, "Omit per-file token count (legacy)")
	Repomix.Flags().BoolVar(&noStats, "no-stats", false, "Omit statistics footer")
	Repomix.Flags().Int64Var(&maxSize, "max-size", 1024*1024, "Maximum file size in bytes")
	Repomix.Flags().StringVar(&ignoreFile, "ignore-file", "", "Path to additional ignore file")
}
