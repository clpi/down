package export

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
	exportCSS    string
)

func findMarkdownFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if info.IsDir() {
			n := info.Name()
			if strings.HasPrefix(n, ".") || n == "node_modules" { return filepath.SkipDir }
			return nil
		}
		if strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func generateHTML(files []string, root string) string {
	var b strings.Builder
	name := filepath.Base(root)

	fmt.Fprintf(&b, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s — Export</title>
<style>
  body { font-family: system-ui, -apple-system, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; line-height: 1.6; color: #1a1a1a; }
  pre { background: #f5f5f5; padding: 1rem; border-radius: 6px; overflow-x: auto; }
  code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
  pre code { background: none; padding: 0; }
  blockquote { border-left: 3px solid #ddd; margin: 0; padding-left: 1rem; color: #555; }
  table { border-collapse: collapse; width: 100%%; }
  th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
  th { background: #f5f5f5; }
  hr { border: none; border-top: 1px solid #eee; margin: 2rem 0; }
  a { color: #2563eb; text-decoration: none; }
  a:hover { text-decoration: underline; }
  .nav { margin-bottom: 2rem; padding-bottom: 1rem; border-bottom: 1px solid #eee; }
  .nav a { margin-right: 1rem; }
  .frontmatter { background: #fafafa; padding: 1rem; border-radius: 6px; margin-bottom: 2rem; font-size: 0.9em; color: #666; }
</style>
</head>
<body>
<div class="nav">
  <strong>%s</strong>
  <span style="color:#999;margin-left:1rem">%d notes · %s</span>
</div>
`, name, name, len(files), time.Now().Format("2006-01-02 15:04"))

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil { continue }
		content := string(data)
		rel, _ := filepath.Rel(root, file)

		// Extract frontmatter
		var fm string
		if strings.HasPrefix(content, "---\n") {
			end := strings.Index(content[4:], "\n---\n")
			if end > 0 {
				fm = content[4 : 4+end]
				content = content[4+end+5:]
			}
		}

		fmt.Fprintf(&b, "<article>\n")

		if fm != "" {
			fmt.Fprintf(&b, "<div class=\"frontmatter\">\n")
			for _, line := range strings.Split(fm, "\n") {
				line = strings.TrimSpace(line)
				if k, v, ok := strings.Cut(line, ":"); ok {
					fmt.Fprintf(&b, "<strong>%s:</strong> %s<br>\n", strings.TrimSpace(k), strings.TrimSpace(v))
				}
			}
			fmt.Fprintf(&b, "</div>\n")
		}

		// Simple markdown to HTML
		html := markdownToHTML(content)
		fmt.Fprintf(&b, "%s\n", html)
		fmt.Fprintf(&b, "<hr>\n")
		fmt.Fprintf(&b, "<small><a href=\"%s\">%s</a></small>\n", rel, rel)
		fmt.Fprintf(&b, "</article>\n\n")
	}

	fmt.Fprintf(&b, "</body>\n</html>\n")
	return b.String()
}

func markdownToHTML(md string) string {
	var b strings.Builder
	lines := strings.Split(md, "\n")
	inCode := false
	inList := false
	inBlockquote := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Fenced code blocks
		if strings.HasPrefix(line, "```") {
			if inCode {
				fmt.Fprintf(&b, "</code></pre>\n")
				inCode = false
			} else {
				if inList { fmt.Fprintf(&b, "</ul>\n"); inList = false }
				if inBlockquote { fmt.Fprintf(&b, "</blockquote>\n"); inBlockquote = false }
				fmt.Fprintf(&b, "<pre><code>")
				inCode = true
			}
			continue
		}
		if inCode {
			fmt.Fprintf(&b, "%s\n", line)
			continue
		}

		// Headings
		if strings.HasPrefix(line, "###### ") {
			fmt.Fprintf(&b, "<h6>%s</h6>\n", line[7:])
			continue
		} else if strings.HasPrefix(line, "##### ") {
			fmt.Fprintf(&b, "<h5>%s</h5>\n", line[6:])
			continue
		} else if strings.HasPrefix(line, "#### ") {
			fmt.Fprintf(&b, "<h4>%s</h4>\n", line[5:])
			continue
		} else if strings.HasPrefix(line, "### ") {
			fmt.Fprintf(&b, "<h3>%s</h3>\n", line[4:])
			continue
		} else if strings.HasPrefix(line, "## ") {
			fmt.Fprintf(&b, "<h2>%s</h2>\n", line[3:])
			continue
		} else if strings.HasPrefix(line, "# ") {
			fmt.Fprintf(&b, "<h1>%s</h1>\n", line[2:])
			continue
		}

		// Horizontal rule
		if strings.TrimSpace(line) == "---" || strings.TrimSpace(line) == "***" {
			fmt.Fprintf(&b, "<hr>\n")
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			if !inBlockquote { fmt.Fprintf(&b, "<blockquote>\n"); inBlockquote = true }
			fmt.Fprintf(&b, "%s<br>\n", inlineFormat(line[2:]))
			continue
		} else if inBlockquote {
			fmt.Fprintf(&b, "</blockquote>\n")
			inBlockquote = false
		}

		// Unordered list
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if !inList { fmt.Fprintf(&b, "<ul>\n"); inList = true }
			content := line[2:]
			// Task list
			if strings.HasPrefix(content, "[ ] ") {
				fmt.Fprintf(&b, "<li><input type=\"checkbox\"> %s</li>\n", inlineFormat(content[4:]))
			} else if strings.HasPrefix(content, "[x] ") || strings.HasPrefix(content, "[X] ") {
				fmt.Fprintf(&b, "<li><input type=\"checkbox\" checked> %s</li>\n", inlineFormat(content[4:]))
			} else {
				fmt.Fprintf(&b, "<li>%s</li>\n", inlineFormat(content))
			}
			continue
		} else if inList && line == "" {
			// Keep list open across blank lines
			continue
		} else if inList {
			fmt.Fprintf(&b, "</ul>\n")
			inList = false
		}

		// Table
		if strings.Contains(line, "|") {
			cols := strings.Split(line, "|")
			realCols := []string{}
			for _, c := range cols {
				if strings.TrimSpace(c) != "" { realCols = append(realCols, strings.TrimSpace(c)) }
			}
			if len(realCols) > 1 {
				if i+1 < len(lines) && strings.Contains(lines[i+1], "---") {
					fmt.Fprintf(&b, "<table>\n<tr>")
					for _, c := range realCols { fmt.Fprintf(&b, "<th>%s</th>", inlineFormat(c)) }
					fmt.Fprintf(&b, "</tr>\n")
					i++ // Skip separator
					for i+1 < len(lines) && strings.Contains(lines[i+1], "|") {
						i++
						cols := strings.Split(lines[i], "|")
						rc := []string{}
						for _, c := range cols { if t := strings.TrimSpace(c); t != "" { rc = append(rc, t) } }
						if len(rc) > 1 {
							fmt.Fprintf(&b, "<tr>")
							for _, c := range rc { fmt.Fprintf(&b, "<td>%s</td>", inlineFormat(c)) }
							fmt.Fprintf(&b, "</tr>\n")
						}
					}
					fmt.Fprintf(&b, "</table>\n")
					continue
				}
			}
		}

		// Paragraph
		if line == "" {
			fmt.Fprintf(&b, "<p></p>\n")
		} else {
			fmt.Fprintf(&b, "<p>%s</p>\n", inlineFormat(line))
		}
	}

	if inList { fmt.Fprintf(&b, "</ul>\n") }
	if inCode { fmt.Fprintf(&b, "</code></pre>\n") }
	if inBlockquote { fmt.Fprintf(&b, "</blockquote>\n") }

	return b.String()
}

func inlineFormat(text string) string {
	text = reBold.ReplaceAllString(text, "<strong>$1</strong>")
	text = reItalic.ReplaceAllString(text, "<em>$1</em>")
	text = reCode.ReplaceAllString(text, "<code>$1</code>")
	text = reLink.ReplaceAllString(text, `<a href="$2">$1</a>`)
	text = strings.ReplaceAll(text, "--", "—")
	return text
}

var (
	reBold   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic = regexp.MustCompile(`\*([^*]+)\*`)
	reCode   = regexp.MustCompile("`([^`]+)`")
	reLink   = regexp.MustCompile(`\[(.+?)\]\((.+?)\)`)
)

func exportHTML(files []string, root, output string) error {
	html := generateHTML(files, root)
	if output == "" {
		fmt.Print(html)
		return nil
	}
	return os.WriteFile(output, []byte(html), 0644)
}

func exportCSV(files []string, root, output string) error {
	// Export all database tables as CSV
	var b strings.Builder
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil { continue }
		lines := strings.Split(string(data), "\n")
		for i := 0; i < len(lines); i++ {
			if strings.Contains(lines[i], "|") && i+1 < len(lines) && strings.Contains(lines[i+1], "---") {
				fmt.Fprintf(&b, "# %s\n", file)
				// Headers
				cols := splitTableRow(lines[i])
				fmt.Fprintf(&b, "%s\n", strings.Join(cols, ","))
				i++ // Skip separator
				for i+1 < len(lines) && strings.Contains(lines[i+1], "|") {
					i++
					cols := splitTableRow(lines[i])
					for j := range cols {
						cols[j] = fmt.Sprintf("%q", strings.Trim(cols[j], `"`))
					}
					fmt.Fprintf(&b, "%s\n", strings.Join(cols, ","))
				}
			}
		}
	}
	if output == "" {
		fmt.Print(b.String())
		return nil
	}
	return os.WriteFile(output, []byte(b.String()), 0644)
}

func splitTableRow(line string) []string {
	parts := strings.Split(line, "|")
	var cols []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" { cols = append(cols, t) }
	}
	return cols
}

func exportMarkdown(files []string, root, output string) error {
	var b strings.Builder
	name := filepath.Base(root)
	fmt.Fprintf(&b, "# %s — Export\n\n> %d notes · %s\n\n", name, len(files), time.Now().Format("2006-01-02 15:04"))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil { continue }
		rel, _ := filepath.Rel(root, file)
		fmt.Fprintf(&b, "## %s\n\n%s\n\n---\n\n", rel, string(data))
	}
	if output == "" {
		fmt.Print(b.String())
		return nil
	}
	return os.WriteFile(output, []byte(b.String()), 0644)
}

var Export = cobra.Command{
	Use:     "export [directory]",
	Aliases: []string{"exp", "ex"},
	Short:   "Export workspace to HTML, CSV, or merged markdown",
	Long:    "Export all markdown notes from a workspace to a single HTML page, CSV file, or merged markdown document. Optionally converts to PDF using pandoc if installed.",
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 { root = args[0] }

		files := findMarkdownFiles(root)
		if len(files) == 0 {
			fmt.Println("No markdown files found")
			return
		}

		switch exportFormat {
		case "html":
			if err := exportHTML(files, root, exportOutput); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if exportOutput != "" {
				fmt.Printf("Exported %d notes to %s\n", len(files), exportOutput)
			}
		case "csv":
			if err := exportCSV(files, root, exportOutput); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if exportOutput != "" {
				fmt.Printf("Exported tables to %s\n", exportOutput)
			}
		case "pdf":
			htmlPath := exportOutput + ".html"
			if htmlPath == ".html" { htmlPath = root + "/export.html" }
			if err := exportHTML(files, root, htmlPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			// Try pandoc conversion
			cmd := exec.Command("pandoc", htmlPath, "-o", exportOutput)
			if err := cmd.Run(); err != nil {
				fmt.Printf("HTML generated at %s\n(Install pandoc for PDF: brew install pandoc)\n", htmlPath)
			} else {
				os.Remove(htmlPath)
				fmt.Printf("Exported %d notes to %s\n", len(files), exportOutput)
			}
		default: // markdown
			if err := exportMarkdown(files, root, exportOutput); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if exportOutput != "" {
				fmt.Printf("Exported %d notes to %s\n", len(files), exportOutput)
			}
		}
	},
}

func init() {
	Export.Flags().StringVarP(&exportFormat, "format", "f", "markdown", "Export format: markdown, html, csv, pdf")
	Export.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path")
	Export.Flags().StringVar(&exportCSS, "css", "", "Custom CSS file for HTML export")
}
