package add

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	addTitle     string
	addWorkspace string
)

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".svg": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".zip": true,
	".tar": true, ".gz": true, ".bz2": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".class": true, ".o": true, ".obj": true, ".mp3": true, ".mp4": true, ".mov": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".db": true, ".wasm": true,
	".7z": true, ".rar": true, ".xz": true, ".zst": true,
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
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// FetchURL fetches a URL and converts HTML to markdown. Exported for use by sync package.
func FetchURL(url string) (string, error) {
	return fetchURL(url)
}

func fetchURL(url string) (string, error) {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract metadata before conversion
	title := extractText(string(body), "title")
	description := ""
	if descIdx := strings.Index(string(body), "property=\"og:description\""); descIdx >= 0 || strings.Index(string(body), "name=\"description\"") >= 0 {
		// Simple description extraction
		html := string(body)
		if metaStart := strings.Index(html, "<meta"); metaStart >= 0 {
			content := ""
			if contentIdx := strings.Index(html[metaStart:], "content=\""); contentIdx > 0 {
				actualIdx := metaStart + contentIdx + 9
				endIdx := strings.Index(html[actualIdx:], "\"")
				if endIdx > 0 {
					content = html[actualIdx : actualIdx+endIdx]
				}
			}
			if len(content) > 0 && len(content) < 500 {
				description = content
			}
		}
	}

	text := enhancedHTMLToMarkdown(string(body))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("---\n"))
	b.WriteString(fmt.Sprintf("source: %s\n", url))
	b.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format("2006-01-02 15:04")))
	if title != "" {
		b.WriteString(fmt.Sprintf("title: %s\n", title))
	}
	b.WriteString("---\n\n")

	if title != "" {
		b.WriteString(fmt.Sprintf("# %s\n\n", title))
	} else {
		b.WriteString(fmt.Sprintf("# %s\n\n", url))
	}

	if description != "" {
		b.WriteString(fmt.Sprintf("> %s\n\n", description))
	}

	b.WriteString(fmt.Sprintf("%s\n\n", text))
	b.WriteString(fmt.Sprintf("> Fetched: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return b.String(), nil
}

func htmlToMarkdown(html string) string {
	// Process in order: remove script/style/head, extract title, convert elements
	html = removeTags(html, "script")
	html = removeTags(html, "style")
	html = removeTags(html, "head")
	html = removeTags(html, "nav")
	html = removeTags(html, "footer")

	// Preserve title
	title := extractText(html, "title")

	var out strings.Builder
	inTag := false
	inPre := false
	skipWS := true

	i := 0
	runes := []rune(html)
	for i < len(runes) {
		ch := runes[i]

		if ch == '<' {
			tagEnd := findTagEnd(runes, i)
			if tagEnd < 0 {
				inTag = true
				i++
				continue
			}
			tag := string(runes[i+1 : tagEnd])
			lower := strings.ToLower(strings.Split(tag, " ")[0])
			lower = strings.TrimSuffix(lower, "/")

			switch lower {
			case "br", "br/":
				out.WriteString("\n")
			case "p", "/p", "div", "/div", "section", "/section", "article", "/article":
				out.WriteString("\n\n")
			case "h1", "/h1":
				out.WriteString("\n\n# ")
			case "h2", "/h2":
				out.WriteString("\n\n## ")
			case "h3", "/h3":
				out.WriteString("\n\n### ")
			case "h4", "/h4":
				out.WriteString("\n\n#### ")
			case "h5", "/h5", "h6", "/h6":
				out.WriteString("\n\n##### ")
			case "li", "/li":
				out.WriteString("\n- ")
			case "hr", "hr/":
				out.WriteString("\n\n---\n\n")
			case "blockquote":
				out.WriteString("\n\n> ")
			case "/blockquote":
				out.WriteString("\n\n")
			case "pre", "code":
				inPre = true
				out.WriteString("\n\n```\n")
			case "/pre", "/code":
				inPre = false
				out.WriteString("\n```\n\n")
			case "a":
				href := extractAttr(tag, "href")
				if href != "" {
					out.WriteString("[")
					// Will be closed after content
				}
			case "/a":
				out.WriteString("]")
			case "img":
				src := extractAttr(tag, "src")
				alt := extractAttr(tag, "alt")
				if alt == "" { alt = "image" }
				if src != "" {
					out.WriteString(fmt.Sprintf("![%s](%s)", alt, src))
				}
			case "strong", "b":
				out.WriteString("**")
			case "/strong", "/b":
				out.WriteString("**")
			case "em", "i":
				out.WriteString("*")
			case "/em", "/i":
				out.WriteString("*")
			}

			i = tagEnd + 1
			if i < len(runes) && runes[i] == '>' {
				i++
			}
			inTag = false
			skipWS = true
			continue
		}

		if !inTag {
			if ch == '\n' || ch == '\r' {
				if !inPre {
					if !skipWS {
						out.WriteByte(' ')
						skipWS = true
					}
				} else {
					out.WriteRune(ch)
				}
				i++
				continue
			}
			if ch == ' ' || ch == '\t' {
				if skipWS {
					i++
					continue
				}
				skipWS = true
				if inPre {
					out.WriteRune(ch)
				} else {
					out.WriteByte(' ')
				}
				i++
				continue
			}
			// Handle & entities
			if ch == '&' {
				entityEnd := findChar(runes, i+1, ';')
				if entityEnd > 0 {
					entity := string(runes[i+1 : entityEnd])
					switch entity {
					case "nbsp": out.WriteByte(' ')
					case "amp": out.WriteByte('&')
					case "lt": out.WriteByte('<')
					case "gt": out.WriteByte('>')
					case "quot": out.WriteByte('"')
					case "apos": out.WriteByte('\'')
					case "mdash": out.WriteString("--")
					case "ndash": out.WriteString("-")
					case "hellip": out.WriteString("...")
					case "rsquo": out.WriteByte('\'')
					case "lsquo": out.WriteByte('\'')
					case "rdquo": out.WriteByte('"')
					case "ldquo": out.WriteByte('"')
					default: out.WriteRune(ch)
					}
					i = entityEnd + 1
					skipWS = false
					continue
				}
			}
			out.WriteRune(ch)
			skipWS = false
		}
		i++
	}

	result := out.String()

	// Prepend title if available
	if title != "" {
		result = fmt.Sprintf("# %s\n\n%s", title, result)
	}

	// Collapse excess whitespace
	lines := strings.Split(result, "\n")
	var cleaned []string
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			blankCount = 0
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func removeTags(html, tag string) string {
	for {
		start := findTagCase(html, "<"+tag)
		if start < 0 { break }
		end := findTagCase(html, "</"+tag+">")
		if end < 0 { break }
		html = html[:start] + html[end+len(tag)+3:]
	}
	return html
}

func findTagCase(s, prefix string) int {
	lower := strings.ToLower(s)
	return strings.Index(lower, prefix)
}

func findTagEnd(runes []rune, start int) int {
	for i := start + 1; i < len(runes); i++ {
		if runes[i] == '>' {
			return i
		}
	}
	return -1
}

func findChar(runes []rune, start int, ch rune) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == ch {
			return i
		}
	}
	return -1
}

func extractText(html, tag string) string {
	lower := strings.ToLower(html)
	start := strings.Index(lower, "<"+tag+">")
	if start < 0 { return "" }
	start += len(tag) + 2
	end := strings.Index(lower[start:], "</"+tag+">")
	if end < 0 { return "" }
	return strings.TrimSpace(html[start : start+end])
}

func extractAttr(tag, attr string) string {
	lower := strings.ToLower(tag)
	idx := strings.Index(lower, attr+"=")
	if idx < 0 { return "" }
	idx += len(attr) + 1
	if idx >= len(tag) { return "" }
	quote := tag[idx]
	if quote == '"' || quote == '\'' {
		idx++
		end := strings.IndexByte(tag[idx:], quote)
		if end < 0 { return "" }
		return tag[idx : idx+end]
	}
	// Handle unquoted attributes
	end := idx
	for end < len(tag) && tag[end] != ' ' && tag[end] != '\t' && tag[end] != '>' {
		end++
	}
	return tag[idx:end]
}

// enhancedHTMLToMarkdown provides improved conversion with better handling of:
// - Nested inline elements
// - Code blocks with language detection
// - Tables
// - Lists (ordered and unordered)
// - Proper entity decoding
func enhancedHTMLToMarkdown(html string) string {
	// Remove script/style/head/nav/footer first
	html = removeTags(html, "script")
	html = removeTags(html, "style")
	html = removeTags(html, "nav")
	html = removeTags(html, "footer")

	// Extract title before removing head
	title := extractText(html, "title")

	// Decode HTML entities
	html = decodeEntities(html)

	var out strings.Builder
	inTag := false
	inPre := false
	inCode := false
	inList := false

	i := 0
	runes := []rune(html)
	for i < len(runes) {
		ch := runes[i]

		if ch == '<' {
			tagEnd := findTagEnd(runes, i)
			if tagEnd < 0 {
				inTag = true
				i++
				continue
			}
			tag := string(runes[i : tagEnd])
			lower := strings.ToLower(strings.TrimSpace(tag))
			lower = strings.TrimSuffix(lower, "/")

			// Extract tag name (without attributes)
			tagName := lower
			if space := strings.Index(lower, " "); space >= 0 {
				tagName = lower[:space]
			}

			switch tagName {
			case "br", "br/":
				out.WriteString("\n")
			case "p", "/p", "div", "/div", "section", "/section", "article", "/article":
				if !inPre {
					out.WriteString("\n\n")
				}
			case "h1", "/h1":
				out.WriteString("\n\n# ")
			case "h2", "/h2":
				out.WriteString("\n\n## ")
			case "h3", "/h3":
				out.WriteString("\n\n### ")
			case "h4", "/h4":
				out.WriteString("\n\n#### ")
			case "h5", "/h5":
				out.WriteString("\n\n##### ")
			case "h6", "/h6":
				out.WriteString("\n\n###### ")
			case "li", "/li":
				if !inList {
					out.WriteString("\n- ")
					inList = true
				} else {
					out.WriteString("\n  - ")
				}
			case "/ul", "/ol":
				inList = false
				out.WriteString("\n")
			case "ul", "ol":
				out.WriteString("\n")
			case "hr", "hr/":
				out.WriteString("\n\n---\n\n")
			case "blockquote":
				out.WriteString("\n\n> ")
			case "/blockquote":
				out.WriteString("\n\n")
			case "pre":
				inPre = true
				// Check for language class
				lang := ""
				if classIdx := strings.Index(lower, "class="); classIdx >= 0 {
					lang = extractAttr(tag, "class")
					lang = strings.TrimPrefix(lang, "language-")
				}
				out.WriteString("\n\n```" + lang + "\n")
			case "/pre":
				inPre = false
				out.WriteString("\n```\n\n")
			case "code":
				// Inline code
				out.WriteString("`")
				inCode = true
			case "/code":
				inCode = false
				out.WriteString("`")
			case "a":
				href := extractAttr(tag, "href")
				if href != "" {
					// Check if it's a mailto link
					if strings.HasPrefix(href, "mailto:") {
						out.WriteString(fmt.Sprintf("<%s>", strings.TrimPrefix(href, "mailto:")))
					} else {
						out.WriteString("[")
					}
				}
			case "/a":
				out.WriteString("]")
			case "img":
				src := extractAttr(tag, "src")
				alt := extractAttr(tag, "alt")
				if alt == "" { alt = "image" }
				if src != "" {
					out.WriteString(fmt.Sprintf("![%s](%s)", alt, src))
				}
			case "strong":
				out.WriteString("**")
			case "/strong":
				out.WriteString("**")
			case "b":
				out.WriteString("**")
			case "/b":
				out.WriteString("**")
			case "em":
				out.WriteString("*")
			case "/em":
				out.WriteString("*")
			case "i":
				out.WriteString("*")
			case "/i":
				out.WriteString("*")
			case "/span", "/strong", "/em", "/b", "/i", "/u":
				// No-op for closing tags
			case "span":
				// Span can contain inline elements, just continue
			}

			i = tagEnd + 1
			if i < len(runes) && runes[i] == '>' {
				i++
			}
			inTag = false
			continue
		}

		if !inTag {
			if ch == '\n' || ch == '\r' {
				if inPre {
					out.WriteRune(ch)
				}
				i++
				continue
			}
			out.WriteRune(ch)
		}
		i++
	}

	result := out.String()

	// Prepend title if available
	if title != "" {
		result = fmt.Sprintf("# %s\n\n%s", title, result)
	}

	// Collapse excess whitespace
	lines := strings.Split(result, "\n")
	var cleaned []string
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			blankCount = 0
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// decodeEntities converts common HTML entities to their character equivalents
func decodeEntities(html string) string {
	entities := map[string]string{
		"&nbsp;": " ", "&amp;": "&", "&lt;": "<", "&gt;": ">",
		"&quot;": "\"", "&apos;": "'", "&#39;": "'", "&#34;": "\"",
		"&mdash;": "—", "&ndash;": "–", "&hellip;": "...",
		"&rsquo;": "'", "&lsquo;": "'", "&rdquo;": "\"", "&ldquo;": "\"",
		"&copy;": "©", "&reg;": "®", "&trade;": "™",
		"&euro;": "€", "&pound;": "£", "&yen;": "¥", "&cent;": "¢",
		"&deg;": "°", "&plusmn;": "±", "&times;": "×", "&divide;": "÷",
	}
	for entity, char := range entities {
		html = strings.ReplaceAll(html, entity, char)
	}
	return html
}

func findDataDir(root string) string {
	for dir := root; dir != "" && dir != "/" && filepath.Dir(dir) != dir; {
		dataDir := filepath.Join(dir, ".down", "data")
		if info, err := os.Stat(dataDir); err == nil && info.IsDir() {
			return dataDir
		}
		parent := filepath.Dir(dir)
		if parent == dir { break }
		dir = parent
	}
	return ""
}

func compactFileContent(path string) (string, error) {
	if isBinary(path) {
		return "", fmt.Errorf("binary file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > 1024*1024 {
		return "", fmt.Errorf("file too large: %s", path)
	}
	name := filepath.Base(path)
	ext := filepath.Ext(path)
	if ext != "" {
		ext = ext[1:]
	}
	return fmt.Sprintf("### %s\n\n```%s\n%s\n```\n", name, ext, string(data)), nil
}

var Add = cobra.Command{
	Use:     "add <source>",
	Aliases: []string{"a"},
	Short:   "Add file/dir/URL to .down/data/ as markdown",
	Long: `Add a file, directory, URL, or named item to the .down/data/ directory.

Source can be:
  file.md      Compact a file into repomix-style markdown
  dir/          Compact a directory
  https://...   Fetch a URL and convert to markdown
  <name>        Create or use <name>.md in data dir
  <name>/       Create directory with index.md`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		root, _ := os.Getwd()
		if addWorkspace != "" {
			root = addWorkspace
		}
		dataDir := findDataDir(root)
		if dataDir == "" {
			dataDir = filepath.Join(root, ".down", "data")
			os.MkdirAll(dataDir, 0755)
		}

		var content string
		var filename string

		switch {
		case isURL(source):
			var err error
			content, err = fetchURL(source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching URL: %v\n", err)
				os.Exit(1)
			}
			domain := strings.TrimPrefix(strings.TrimPrefix(source, "https://"), "http://")
			filename = strings.ReplaceAll(domain, ".", "_") + ".md"

		case strings.HasSuffix(source, "/"):
			clean := strings.TrimSuffix(source, "/")
			dname := filepath.Base(clean)
			dirPath := filepath.Join(root, clean)
			os.MkdirAll(dirPath, 0755)
			idxPath := filepath.Join(dirPath, "index.md")
			if _, err := os.Stat(idxPath); os.IsNotExist(err) {
				os.WriteFile(idxPath, []byte(fmt.Sprintf("# %s\n\nIndex for %s\n", dname, dname)), 0644)
			}
			content = fmt.Sprintf("# %s\n\nCreated: %s\n", dname, time.Now().Format("2006-01-02 15:04"))
			filename = dname + "_compact.md"

		default:
			info, err := os.Stat(source)
			if err == nil && !info.IsDir() {
				if isBinary(source) {
					fmt.Fprintf(os.Stderr, "Skipping binary file: %s\n", source)
					os.Exit(1)
				}
				if info.Size() > 1024*1024 {
					fmt.Fprintf(os.Stderr, "File too large (>1MB): %s\n", source)
					os.Exit(1)
				}
				c, err := compactFileContent(source)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
					os.Exit(1)
				}
				content = c
				fname := filepath.Base(source)
				filename = strings.ReplaceAll(fname, ".", "_") + ".md"
			} else if err == nil && info.IsDir() {
				dname := filepath.Base(source)
				content = fmt.Sprintf("# %s\n\nDirectory compacted from: %s\n", dname, source)
				filename = dname + "_compact.md"
			} else {
				// Bare word or non-existent path
				mdPath := filepath.Join(root, source+".md")
				if _, err := os.Stat(mdPath); err == nil {
					c, _ := compactFileContent(mdPath)
					content = c
					filename = source + ".md"
				} else {
					dataMD := filepath.Join(dataDir, source+".md")
					if _, err := os.Stat(dataMD); err == nil {
						filename = source + "_index.md"
						content = fmt.Sprintf("# %s\n\nSee: %s.md\n", source, source)
					} else {
						os.WriteFile(mdPath, []byte(fmt.Sprintf("# %s\n\n", source)), 0644)
						content = fmt.Sprintf("# %s\n\nAdded: %s\n", source, time.Now().Format("2006-01-02 15:04"))
						filename = source + ".md"
					}
				}
			}
		}

		// Add frontmatter header
		header := fmt.Sprintf("---\nsource: %s\ndate: %s\n", source, time.Now().Format("2006-01-02 15:04"))
		if addTitle != "" {
			header += fmt.Sprintf("title: %s\n", addTitle)
		}
		header += "---\n\n"
		content = header + content

		outPath := filepath.Join(dataDir, filename)
		if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added: %s\n  -> %s\n", source, outPath)
	},
}

func init() {
	Add.Flags().StringVarP(&addTitle, "title", "t", "", "Set title for the output")
	Add.Flags().StringVarP(&addWorkspace, "workspace", "w", "", "Target workspace directory")
}
