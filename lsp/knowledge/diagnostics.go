package knowledge

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Diagnostic struct {
	Line     int
	ColStart int
	ColEnd   int
	Message  string
	Severity DiagSeverity
}

type DiagSeverity int

const (
	SeverityError   DiagSeverity = 1
	SeverityWarning DiagSeverity = 2
	SeverityInfo    DiagSeverity = 3
	SeverityHint    DiagSeverity = 4
)

var (
	reDiagWikiLink = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reDiagMdLink   = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	reDiagTask     = regexp.MustCompile(`^(\s*)[-*]\s+\[( )\]\s+(.+)`)
)

func DiagnoseDocument(g *Graph, uri string, text string) []Diagnostic {
	var diags []Diagnostic
	lines := strings.Split(text, "\n")
	docDir := filepath.Dir(cleanURI(uri))

	inCode := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}

		for _, m := range reDiagWikiLink.FindAllStringSubmatchIndex(line, -1) {
			nameStart, nameEnd := m[2], m[3]
			target := line[nameStart:nameEnd]
			parts := strings.SplitN(target, "|", 2)
			linkTarget := strings.TrimSpace(parts[0])

			if !wikiLinkResolvable(g, docDir, uri, linkTarget) {
				diags = append(diags, Diagnostic{
					Line:     i,
					ColStart: m[0],
					ColEnd:   m[1],
					Message:  "Unresolved wiki link: " + linkTarget,
					Severity: SeverityWarning,
				})
			}
		}

		for _, m := range reDiagMdLink.FindAllStringSubmatchIndex(line, -1) {
			hrefStart, hrefEnd := m[4], m[5]
			href := line[hrefStart:hrefEnd]

			if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") {
				continue
			}
			resolved := filepath.Join(docDir, href)
			if _, err := os.Stat(resolved); os.IsNotExist(err) {
				diags = append(diags, Diagnostic{
					Line:     i,
					ColStart: m[0],
					ColEnd:   m[1],
					Message:  "Broken link: " + href + " (file not found)",
					Severity: SeverityWarning,
				})
			}
		}

		if m := reDiagTask.FindStringSubmatch(line); m != nil {
			taskText := m[3]
			if containsOverdueDate(taskText) {
				diags = append(diags, Diagnostic{
					Line:     i,
					ColStart: 0,
					ColEnd:   len(line),
					Message:  "Open task may be overdue",
					Severity: SeverityInfo,
				})
			}
		}
	}

	return diags
}

func wikiLinkResolvable(g *Graph, docDir string, docURI string, target string) bool {
	candidates := []string{
		filepath.Join(docDir, target+".md"),
		filepath.Join(docDir, target+".markdown"),
		filepath.Join(docDir, target),
		filepath.Join(docDir, strings.ReplaceAll(target, " ", "-")+".md"),
		filepath.Join(docDir, strings.ReplaceAll(target, " ", "_")+".md"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return true
		}
	}

	for _, kind := range []EntityKind{KindDocument, KindConcept, KindProject} {
		id := entityID(kind, target)
		if ent, ok := g.Entities[id]; ok {
			for _, src := range ent.Sources {
				if src.URI != docURI {
					return true
				}
			}
		}
	}
	return false
}

func containsOverdueDate(text string) bool {
	matches := reDate.FindAllString(text, -1)
	return len(matches) > 0
}
