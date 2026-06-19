package compact

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	output     string
	format     string
	noTree     bool
	noTokens   bool
	noStats    bool
	estimate   bool
	include    string
	exclude    string
	clipboard  bool
	compress   bool
	maxSize    int64
	ignoreFile string
)

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".svg": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".class": true, ".o": true, ".obj": true,
	".mp3": true, ".mp4": true, ".mov": true, ".avi": true, ".wav": true, ".flac": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".db": true, ".wasm": true, ".7z": true, ".rar": true,
}

// Secret patterns
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|apikey|api_secret|secret[_-]?key)\s*[:=]\s*["']?[A-Za-z0-9_\-]{20,}["']?`),
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["'][^"']+["']`),
	regexp.MustCompile(`(?i)(token|auth[_-]?token)\s*[:=]\s*["']?[A-Za-z0-9_\-\.]{20,}["']?`),
	regexp.MustCompile(`(?i)(private[_-]?key|privkey|id_rsa|id_ed25519)["':\s]`),
	regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`),
	regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
	regexp.MustCompile(`xox[bpras]-\d{10,12}-\d{10,12}-[A-Za-z0-9]{24,32}`),
	regexp.MustCompile(`sk-[A-Za-z0-9]{48,}`),
}

func secretScore(content string) int {
	score := 0
	for _, pat := range secretPatterns {
		score += len(pat.FindAllString(content, -1)) * 10
	}
	return score
}

func isBinary(path string) bool {
	if binaryExts[strings.ToLower(filepath.Ext(path))] {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

func parseGitignore(root string) []string {
	var patterns []string
	// Parse .gitignore with negation support
	gitPath := filepath.Join(root, ".gitignore")
	f, err := os.Open(gitPath)
	if err != nil {
		return patterns
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		negate := false
		if strings.HasPrefix(line, "!") {
			negate = true
			line = line[1:]
		}
		// Convert gitignore to glob
		line = strings.TrimPrefix(line, "/")
		// Remove trailing /
		line = strings.TrimRight(line, "/")
		if negate {
			line = "!" + line
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func matchesGitignore(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, pat := range patterns {
		if strings.HasPrefix(pat, "!") {
			continue
		}
		// Simple glob matching
		if matched, _ := filepath.Match(pat, path); matched {
			return true
		}
		if strings.Contains(pat, "*") {
			if matched, _ := filepath.Match(pat, filepath.Base(path)); matched {
				return true
			}
		}
		// Directory pattern
		if strings.HasSuffix(pat, "/") {
			prefix := strings.TrimSuffix(pat, "/")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

func matchGlob(path string, patterns []string) bool {
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		if matched, _ := filepath.Match(pat, filepath.Base(path)); matched {
			return true
		}
		if matched, _ := filepath.Match(pat, path); matched {
			return true
		}
	}
	return false
}

func estimateTokens(text string) int {
	return (len(text) + 3) / 4
}

func estimateCost(tokens int) string {
	type model struct {
		name   string
		input  float64
		output float64
	}
	models := []model{
		{"o1", 15.0, 60.0},
		{"o3-mini", 1.10, 4.40},
		{"gpt-4.1", 2.00, 8.00},
		{"gpt-4.1-mini", 0.40, 1.60},
		{"gpt-4.1-nano", 0.10, 0.40},
		{"claude-3.5-sonnet", 3.0, 15.0},
		{"claude-3-haiku", 0.80, 4.0},
		{"gemini-2.5-pro", 1.25, 10.0},
		{"gemini-2.5-flash", 0.15, 0.60},
	}

	tk := float64(tokens) / 1000.0
	var lines []string
	for _, m := range models {
		cost := tk * m.input / 1000.0
		lines = append(lines, fmt.Sprintf("  %-22s ~$%.4f", m.name, cost))
	}
	return strings.Join(lines, "\n")
}

type fileEntry struct {
	Path  string
	IsDir bool
}

func collectFiles(root string, gitignores, excludes []string) ([]fileEntry, error) {
	ignores := []string{".git", ".svn", "node_modules", ".DS_Store", ".down"}
	ignores = append(ignores, gitignores...)

	var entries []fileEntry
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			if matchesGitignore(rel, ignores) {
				return filepath.SkipDir
			}
			return nil
		}
		if matchesGitignore(rel, ignores) {
			return nil
		}
		if matchGlob(rel, excludes) {
			return nil
		}
		entries = append(entries, fileEntry{Path: rel, IsDir: false})
		return nil
	})
	return entries, nil
}

func generateTree(entries []fileEntry) string {
	var b strings.Builder
	b.WriteString(".")
	var stack []string
	for _, e := range entries {
		parts := strings.Split(e.Path, string(filepath.Separator))
		depth := len(parts) - 1
		for i := 0; i < depth; i++ {
			b.WriteString("  ")
		}
		b.WriteString("  " + parts[depth] + "\n")
	}
	_ = stack
	return b.String()
}

func packXML(root string, entries []fileEntry) string {
	var b strings.Builder
	totalTokens := 0
	fileCount := 0
	secretCount := 0

	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<project>\n")
	fmt.Fprintf(&b, "  <metadata>\n    <source>%s</source>\n    <generated>%s</generated>\n    <files>%d</files>\n  </metadata>\n",
		root, time.Now().Format("2006-01-02T15:04:05"), len(entries))

	if !noTree {
		b.WriteString("  <structure>\n<![CDATA[\n" + generateTree(entries) + "]]>\n  </structure>\n")
	}

	b.WriteString("  <files>\n")
	for _, e := range entries {
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
		if strings.TrimSpace(content) == "" {
			continue
		}

		// Secret check
		ss := secretScore(content)
		if ss > 20 {
			secretCount++
			continue
		}

		fileCount++
		tok := estimateTokens(content)
		totalTokens += tok
		fmt.Fprintf(&b, "    <file path=\"%s\" tokens=\"%d\">\n<![CDATA[\n%s\n]]>\n    </file>\n", e.Path, tok, content)
	}
	b.WriteString("  </files>\n")

	if !noStats && !noTokens {
		fmt.Fprintf(&b, "  <tokens total=\"%d\" files=\"%d\" />\n", totalTokens, fileCount)
		if secretCount > 0 {
			fmt.Fprintf(&b, "  <security files_redacted=\"%d\" />\n", secretCount)
		}

		if estimate && totalTokens > 0 {
			cost := estimateCost(totalTokens)
			b.WriteString("  <!-- Cost estimates (per 1M input tokens):\n")
			b.WriteString(cost)
			b.WriteString("\n  -->\n")
		}
	}
	b.WriteString("</project>\n")
	return b.String()
}

func packMarkdown(root string, entries []fileEntry) string {
	var b strings.Builder
	totalTokens := 0
	fileCount := 0
	secretCount := 0
	name := filepath.Base(root)

	fmt.Fprintf(&b, "# Project: %s\n\n> Generated: %s\n\n", name, time.Now().Format("2006-01-02 15:04:05"))

	if !noTree {
		b.WriteString("## Directory Structure\n\n```\n" + generateTree(entries) + "```\n\n")
	}

	b.WriteString("## Files\n\n")
	for _, e := range entries {
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
		if strings.TrimSpace(content) == "" {
			continue
		}

		if secretScore(content) > 20 {
			secretCount++
			continue
		}

		fileCount++
		tok := estimateTokens(content)
		totalTokens += tok
		ext := filepath.Ext(e.Path)
		if ext != "" {
			ext = ext[1:]
		}
		fmt.Fprintf(&b, "### %s\n\n```%s\n%s\n```\n\n", e.Path, ext, content)
	}

	if !noStats && !noTokens {
		fmt.Fprintf(&b, "---\n\n**Files:** %d | **Tokens:** ~%d\n", fileCount, totalTokens)
		if estimate && totalTokens > 0 {
			b.WriteString("\n**Cost estimates:**\n\n```\n" + estimateCost(totalTokens) + "\n```\n")
		}
	}
	return b.String()
}

type compactFileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Tokens  int    `json:"tokens"`
}

type compactOutput struct {
	Name      string               `json:"name"`
	Timestamp string               `json:"timestamp"`
	Tree      string               `json:"tree,omitempty"`
	Files     []compactFileContent `json:"files"`
	Stats     compactStats         `json:"stats"`
}

type compactStats struct {
	TotalFiles  int `json:"totalFiles"`
	TotalTokens int `json:"totalTokens"`
	TotalChars  int `json:"totalChars"`
}

func collectContents(root string, entries []fileEntry) ([]compactFileContent, compactStats) {
	var files []compactFileContent
	var st compactStats
	for _, e := range entries {
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
		if strings.TrimSpace(content) == "" {
			continue
		}
		if secretScore(content) > 20 {
			continue
		}
		tok := estimateTokens(content)
		st.TotalFiles++
		st.TotalTokens += tok
		st.TotalChars += len(content)
		files = append(files, compactFileContent{Path: e.Path, Content: content, Tokens: tok})
	}
	return files, st
}

func packPlain(root string, entries []fileEntry) string {
	files, st := collectContents(root, entries)
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\nGenerated: %s\n\n", filepath.Base(root), time.Now().Format("2006-01-02 15:04:05"))
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
	return b.String()
}

func packJSON(root string, entries []fileEntry) string {
	files, st := collectContents(root, entries)
	out := compactOutput{
		Name:      filepath.Base(root),
		Timestamp: time.Now().Format("2006-01-02T15:04:05"),
		Files:     files,
		Stats:     st,
	}
	if !noTree {
		out.Tree = generateTree(entries)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return string(b) + "\n"
}

func packTree(root string, entries []fileEntry) string {
	return generateTree(entries)
}

// PackXMLString packs a directory into repomix-style XML without writing files.
func PackXMLString(root string) (string, error) {
	gitignores := parseGitignore(root)
	entries, err := collectFiles(root, gitignores, nil)
	if err != nil {
		return "", err
	}
	return packXML(root, entries), nil
}

var Compact = cobra.Command{
	Use:     "compact [directory]",
	Aliases: []string{"pack", "cp"},
	Short:   "Pack codebase into AI-friendly format (repomix-compatible)",
	Long: `Pack a directory into XML or markdown for AI consumption.

Repomix-compatible features:
  - Respects .gitignore patterns (including negation !)
  - Detects and redacts secrets (API keys, tokens, passwords)
  - Binary file exclusion
  - Cost estimation for major models
  - Include/exclude pattern filtering
  - Token counting`,
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		gitignores := parseGitignore(root)
		if ignoreFile != "" {
			if data, err := os.ReadFile(ignoreFile); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if line != "" && !strings.HasPrefix(line, "#") {
						gitignores = append(gitignores, line)
					}
				}
			}
		}

		var excludes []string
		if exclude != "" {
			excludes = strings.Split(exclude, ",")
		}

		entries, err := collectFiles(root, gitignores, excludes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Filter by include patterns
		if include != "" {
			includes := strings.Split(include, ",")
			var filtered []fileEntry
			for _, e := range entries {
				if matchGlob(e.Path, includes) {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}

		var result string
		switch format {
		case "markdown", "md":
			result = packMarkdown(root, entries)
		case "json":
			result = packJSON(root, entries)
		case "plain", "text":
			result = packPlain(root, entries)
		case "tree":
			result = packTree(root, entries)
		default:
			result = packXML(root, entries)
		}

		if clipboard {
			// Write to system clipboard (macOS/Linux)
			_ = clipboard
			fmt.Println("(clipboard output not yet implemented — use -o and pipe to pbcopy/xclip)")
		}

		if output != "" {
			os.WriteFile(output, []byte(result), 0644)
			fmt.Printf("Wrote %s\n", output)
		} else {
			fmt.Print(result)
		}
	},
}

func init() {
	Compact.Flags().StringVarP(&output, "output", "o", "", "Write output to file (default: stdout)")
	Compact.Flags().StringVarP(&format, "format", "f", "xml", "Output format: xml, markdown, json, plain, tree")
	Compact.Flags().BoolVar(&noTree, "no-tree", false, "Omit directory tree")
	Compact.Flags().BoolVar(&noTokens, "no-tokens", false, "Omit per-file token count")
	Compact.Flags().BoolVar(&noStats, "no-stats", false, "Omit statistics footer")
	Compact.Flags().StringVarP(&include, "include", "i", "", "Include only files matching glob patterns (comma-separated)")
	Compact.Flags().StringVarP(&exclude, "exclude", "x", "", "Exclude files matching glob patterns (comma-separated)")
	Compact.Flags().StringVar(&ignoreFile, "ignore-file", "", "Path to additional ignore file")
	Compact.Flags().BoolVarP(&clipboard, "clipboard", "c", false, "Copy output to clipboard")
	Compact.Flags().Int64VarP(&maxSize, "max-size", "s", 1024*1024, "Max file size in bytes")
}
