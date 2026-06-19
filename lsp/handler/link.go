package handler

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	reLinkWiki = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reLinkMd   = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
)

func (s *State) LinkResolve(_ *glsp.Context, p *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return p, nil
}

func (s *State) Links(_ *glsp.Context, p *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	docDir := filepath.Dir(strings.TrimPrefix(uri, "file://"))
	lines := strings.Split(doc, "\n")
	var links []protocol.DocumentLink

	inCode := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}

		for _, m := range reLinkWiki.FindAllStringSubmatchIndex(line, -1) {
			fullStart, fullEnd := m[0], m[1]
			nameStart, nameEnd := m[2], m[3]
			target := line[nameStart:nameEnd]
			parts := strings.SplitN(target, "|", 2)
			linkTarget := strings.TrimSpace(parts[0])

			resolved := resolveWikiTarget(docDir, linkTarget)
			if resolved != "" {
				targetURI := "file://" + resolved
				tooltip := "Open " + linkTarget
				links = append(links, protocol.DocumentLink{
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullStart)},
						End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullEnd)},
					},
					Target:  &targetURI,
					Tooltip: &tooltip,
				})
			}
		}

		for _, m := range reLinkMd.FindAllStringSubmatchIndex(line, -1) {
			fullStart, fullEnd := m[0], m[1]
			hrefStart, hrefEnd := m[4], m[5]
			href := line[hrefStart:hrefEnd]

			var targetURI string
			if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
				targetURI = href
			} else if strings.HasPrefix(href, "#") {
				continue
			} else {
				resolved := filepath.Join(docDir, href)
				targetURI = "file://" + resolved
			}

			tooltip := href
			links = append(links, protocol.DocumentLink{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullStart)},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullEnd)},
				},
				Target:  &targetURI,
				Tooltip: &tooltip,
			})
		}
	}

	return links, nil
}

func resolveWikiTarget(docDir string, target string) string {
	candidates := []string{
		filepath.Join(docDir, target+".md"),
		filepath.Join(docDir, target+".markdown"),
		filepath.Join(docDir, target),
		filepath.Join(docDir, strings.ReplaceAll(target, " ", "-")+".md"),
		filepath.Join(docDir, strings.ReplaceAll(target, " ", "_")+".md"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}
