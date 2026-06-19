package compact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	output string
	format string
	noTree bool
	noTokens bool
)

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".svg": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".zip": true,
	".tar": true, ".gz": true, ".bz2": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".class": true, ".o": true, ".obj": true, ".mp3": true, ".mp4": true, ".mov": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".db": true, ".wasm": true,
}

var maxFileSize int64 = 1024 * 1024

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
		// Also match against just the file name (not full path)
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

func loadDownIgnore(root string) []string {
	var patterns []string
	ignorePath := filepath.Join(root, ".down", ".downignore")
	data, err := os.ReadFile(ignorePath)
	if err != nil {
		return patterns
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

func collectFiles(root string) ([]fileEntry, error) {
	defaultIgnores := []string{".git", ".svn", "node_modules", ".DS_Store", ".down"}
	customIgnores := loadDownIgnore(root)
	ignores := append(defaultIgnores, customIgnores...)
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

func packXML(root string) (string, error) {
	entries, err := collectFiles(root)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	totalTokens := 0
	fileCount := 0

	fmt.Fprintf(&b, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(&b, "<project>\n")
	fmt.Fprintf(&b, "  <metadata>\n")
	fmt.Fprintf(&b, "    <source>%s</source>\n", root)
	fmt.Fprintf(&b, "    <generated>%s</generated>\n", time.Now().Format("2006-01-02T15:04:05"))
	fmt.Fprintf(&b, "  </metadata>\n")

	if !noTree {
		fmt.Fprintf(&b, "  <structure>\n")
		fmt.Fprintf(&b, "  <![CDATA[\n%s]]>\n", generateTree(entries))
		fmt.Fprintf(&b, "  </structure>\n")
	}

	fmt.Fprintf(&b, "  <files>\n")
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		full := filepath.Join(root, e.Path)
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.Size() > maxFileSize {
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
		fileCount++
		tok := estimateTokens(content)
		totalTokens += tok
		fmt.Fprintf(&b, "    <file path=\"%s\" tokens=\"%d\">\n", e.Path, tok)
		fmt.Fprintf(&b, "<![CDATA[\n%s\n]]>\n", content)
		fmt.Fprintf(&b, "    </file>\n")
	}
	fmt.Fprintf(&b, "  </files>\n")

	if !noTokens {
		fmt.Fprintf(&b, "  <tokens total=\"%d\" files=\"%d\" />\n", totalTokens, fileCount)
	}
	fmt.Fprintf(&b, "</project>\n")
	return b.String(), nil
}

func packMarkdown(root string) (string, error) {
	entries, err := collectFiles(root)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	totalTokens := 0
	fileCount := 0
	name := filepath.Base(root)

	fmt.Fprintf(&b, "# Project: %s\n\n", name)
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	if !noTree {
		fmt.Fprintf(&b, "## Directory Structure\n\n```\n%s```\n\n", generateTree(entries))
	}

	fmt.Fprintf(&b, "## Files\n\n")
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		full := filepath.Join(root, e.Path)
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.Size() > maxFileSize {
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
		fileCount++
		totalTokens += estimateTokens(content)
		ext := filepath.Ext(e.Path)
		if ext != "" {
			ext = ext[1:]
		}
		fmt.Fprintf(&b, "### %s\n\n```%s\n%s\n```\n\n", e.Path, ext, content)
	}

	if !noTokens {
		fmt.Fprintf(&b, "---\n\n**Files:** %d | **Tokens:** ~%d\n\n", fileCount, totalTokens)
	}
	return b.String(), nil
}

var Compact = cobra.Command{
	Use:     "compact [directory]",
	Aliases: []string{"pack", "cp"},
	Short:   "Pack codebase into AI-friendly format",
	Long:    "Pack a directory into XML or markdown for AI consumption, respecting .gitignore and excluding binary files.",
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		var result string
		var err error
		if format == "markdown" {
			result, err = packMarkdown(root)
		} else {
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
	Compact.Flags().StringVarP(&output, "output", "o", "", "Write output to file (default: stdout)")
	Compact.Flags().StringVarP(&format, "format", "f", "xml", "Output format: xml, markdown")
	Compact.Flags().BoolVar(&noTree, "no-tree", false, "Omit directory tree")
	Compact.Flags().BoolVar(&noTokens, "no-tokens", false, "Omit token count")
}
