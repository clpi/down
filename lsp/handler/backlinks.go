package handler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/clpi/down/lsp/knowledge"
)

// Backlink represents a reference from another document to the current one.
type Backlink struct {
	SourceURI   string `json:"sourceUri"`
	SourceTitle string `json:"sourceTitle"`
	Line        int    `json:"line"`
	Context     string `json:"context"` // surrounding text
	Kind        string `json:"kind"`    // "wiki_link", "mention", "tag", "relation"
}

// BacklinksResult is the response from the backlinks command.
type BacklinksResult struct {
	URI       string     `json:"uri"`
	Title     string     `json:"title"`
	Backlinks []Backlink `json:"backlinks"`
	Count     int        `json:"count"`
}

// ComputeBacklinks finds all documents that reference the given document.
func (s *State) ComputeBacklinks(targetURI string) *BacklinksResult {
	if s.Graph == nil {
		return &BacklinksResult{URI: targetURI}
	}

	result := &BacklinksResult{
		URI:       targetURI,
		Backlinks: make([]Backlink, 0),
	}

	// Get the document title from the first heading
	if text, ok := s.Documents[targetURI]; ok {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				result.Title = strings.TrimSpace(trimmed[2:])
				break
			}
		}
	}
	if result.Title == "" {
		result.Title = filepath.Base(strings.TrimPrefix(targetURI, "file://"))
	}

	// Find all entities from the target document
	targetEntities := s.Graph.EntitiesByDocument(targetURI)
	targetNames := make(map[string]bool)
	for _, ent := range targetEntities {
		targetNames[strings.ToLower(ent.Name)] = true
	}

	// Also consider the document name itself
	docName := strings.TrimSuffix(filepath.Base(strings.TrimPrefix(targetURI, "file://")), filepath.Ext(targetURI))
	targetNames[strings.ToLower(docName)] = true

	// Search through all relations pointing to entities in this document
	for _, rel := range s.Graph.Relations {
		// Check if relation points to an entity in target doc
		targetEnt, exists := s.Graph.Entities[rel.To]
		if !exists {
			continue
		}

		isTargetEntity := false
		for _, src := range targetEnt.Sources {
			if src.URI == targetURI {
				isTargetEntity = true
				break
			}
		}
		if !isTargetEntity {
			continue
		}

		// The source of this relation is a backlink
		if rel.Source.URI == targetURI {
			continue // self-reference
		}

		context := getLineContext(s.Documents, rel.Source.URI, rel.Source.Line)
		sourceTitle := getDocTitle(s.Documents, rel.Source.URI)

		kind := "relation"
		switch rel.Kind {
		case knowledge.RelLinksTo:
			kind = "wiki_link"
		case knowledge.RelMentions:
			kind = "mention"
		case knowledge.RelTaggedWith:
			kind = "tag"
		}

		result.Backlinks = append(result.Backlinks, Backlink{
			SourceURI:   rel.Source.URI,
			SourceTitle: sourceTitle,
			Line:        rel.Source.Line,
			Context:     context,
			Kind:        kind,
		})
	}

	// Also scan open documents for direct wiki link references
	for docURI, docText := range s.Documents {
		if docURI == targetURI {
			continue
		}
		lines := strings.Split(docText, "\n")
		for i, line := range lines {
			// Check for wiki links pointing to our document
			for _, m := range reLinkWiki.FindAllStringSubmatch(line, -1) {
				parts := strings.SplitN(m[1], "|", 2)
				linkTarget := strings.TrimSpace(parts[0])
				if strings.EqualFold(linkTarget, docName) || targetNames[strings.ToLower(linkTarget)] {
					// Avoid duplicates
					isDup := false
					for _, existing := range result.Backlinks {
						if existing.SourceURI == docURI && existing.Line == i {
							isDup = true
							break
						}
					}
					if !isDup {
						result.Backlinks = append(result.Backlinks, Backlink{
							SourceURI:   docURI,
							SourceTitle: getDocTitle(s.Documents, docURI),
							Line:        i,
							Context:     strings.TrimSpace(line),
							Kind:        "wiki_link",
						})
					}
				}
			}
		}
	}

	result.Count = len(result.Backlinks)
	return result
}

// BacklinksSummary returns a markdown summary of backlinks for use in hover/commands.
func (s *State) BacklinksSummary(targetURI string) string {
	bl := s.ComputeBacklinks(targetURI)
	if bl.Count == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n\n---\n**Backlinks** (%d):\n", bl.Count))
	for _, link := range bl.Backlinks {
		icon := "📄"
		switch link.Kind {
		case "wiki_link":
			icon = "🔗"
		case "mention":
			icon = "👤"
		case "tag":
			icon = "🏷️"
		}
		sb.WriteString(fmt.Sprintf("- %s **%s** (line %d)\n", icon, link.SourceTitle, link.Line+1))
		if link.Context != "" {
			ctx := link.Context
			if len(ctx) > 80 {
				ctx = ctx[:77] + "..."
			}
			sb.WriteString(fmt.Sprintf("  > %s\n", ctx))
		}
	}
	return sb.String()
}

func getLineContext(docs map[string]string, uri string, line int) string {
	text, ok := docs[uri]
	if !ok {
		return ""
	}
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line])
}

func getDocTitle(docs map[string]string, uri string) string {
	text, ok := docs[uri]
	if !ok {
		return filepath.Base(strings.TrimPrefix(uri, "file://"))
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return filepath.Base(strings.TrimPrefix(uri, "file://"))
}
