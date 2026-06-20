package handler

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/clpi/down/lsp/knowledge"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	reLinkWiki    = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reLinkMd      = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	reLinkEmbed   = regexp.MustCompile(`!\[\[([^\]]+)\]\]`)
	reLinkTag     = regexp.MustCompile(`(?:^|\s)(#[A-Za-z][A-Za-z0-9_/-]*)`)
	reLinkMention = regexp.MustCompile(`(?:^|\s)(@[A-Za-z][A-Za-z0-9_.-]*)`)
)

func (s *State) LinkResolve(_ *glsp.Context, p *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	if p.Target == nil {
		return p, nil
	}
	target := string(*p.Target)
	if strings.HasPrefix(target, "down:") {
		resolved := s.resolveDownURI(target)
		if resolved != "" {
			uri := protocol.DocumentUri(resolved)
			p.Target = &uri
		}
	}
	return p, nil
}

func (s *State) Links(_ *glsp.Context, p *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	docDir := filepath.Dir(cleanURI(uri))
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

			var targetURI string
			if resolved := resolveWikiTarget(docDir, linkTarget); resolved != "" {
				targetURI = "file://" + resolved
			} else if loc := s.firstEntityLocation(linkTarget); loc != nil {
				targetURI = string(loc.URI)
			}
			if targetURI == "" {
				continue
			}
			tooltip := "Wiki link: " + linkTarget
			links = append(links, protocol.DocumentLink{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullStart)},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullEnd)},
				},
				Target:  &targetURI,
				Tooltip: &tooltip,
			})
		}

		for _, m := range reLinkEmbed.FindAllStringSubmatchIndex(line, -1) {
			fullStart, fullEnd := m[0], m[1]
			nameStart, nameEnd := m[2], m[3]
			target := strings.TrimSpace(line[nameStart:nameEnd])
			parts := strings.SplitN(target, "|", 2)
			linkTarget := strings.TrimSpace(parts[0])

			var targetURI string
			if resolved := resolveWikiTarget(docDir, linkTarget); resolved != "" {
				targetURI = "file://" + resolved
			}
			if targetURI == "" {
				continue
			}
			tooltip := "Embed: " + linkTarget
			links = append(links, protocol.DocumentLink{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullStart)},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(fullEnd)},
				},
				Target:  &targetURI,
				Tooltip: &tooltip,
			})
		}

		for _, m := range reLinkMd.FindAllStringSubmatchIndex(line, -1) {
			fullStart, fullEnd := m[0], m[1]
			hrefStart, hrefEnd := m[4], m[5]
			href := line[hrefStart:hrefEnd]

			var targetURI string
			if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
				targetURI = href
			} else if strings.HasPrefix(href, "#") {
				targetURI = uri + href
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

		for _, m := range reLinkTag.FindAllStringSubmatchIndex(line, -1) {
			tagStart, tagEnd := m[2], m[3]
			tag := line[tagStart:tagEnd]
			targetURI := s.tagDocumentURI(tag)
			if targetURI == "" {
				targetURI = "down://tag/" + strings.TrimPrefix(tag, "#")
			}
			tooltip := "Tag: " + tag
			links = append(links, protocol.DocumentLink{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(tagStart)},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(tagEnd)},
				},
				Target:  &targetURI,
				Tooltip: &tooltip,
			})
		}

		for _, m := range reLinkMention.FindAllStringSubmatchIndex(line, -1) {
			mentionStart, mentionEnd := m[2], m[3]
			mention := line[mentionStart:mentionEnd]
			name := strings.TrimPrefix(mention, "@")
			targetURI := s.mentionDocumentURI(name)
			if targetURI == "" {
				targetURI = "down://mention/" + name
			}
			tooltip := "Mention: " + mention
			links = append(links, protocol.DocumentLink{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(mentionStart)},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(mentionEnd)},
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

func (s *State) tagDocumentURI(tag string) string {
	if s.Graph == nil {
		return ""
	}
	name := strings.TrimPrefix(tag, "#")
	loc := s.firstEntityLocation(name)
	if loc == nil {
		return ""
	}
	return string(loc.URI)
}

func (s *State) mentionDocumentURI(name string) string {
	if s.Graph == nil {
		return ""
	}
	loc := s.firstEntityLocation(name)
	if loc == nil {
		return ""
	}
	return string(loc.URI)
}

func (s *State) firstEntityLocation(name string) *protocol.Location {
	if s.Graph == nil || name == "" {
		return nil
	}
	for _, ent := range s.Graph.Search(name) {
		if !strings.EqualFold(ent.Name, name) {
			continue
		}
		for _, src := range ent.Sources {
			return &protocol.Location{
				URI: protocol.DocumentUri(src.URI),
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(src.Line), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(src.Line), Character: protocol.UInteger(len(ent.Name))},
				},
			}
		}
	}
	return nil
}

func (s *State) resolveDownURI(uri string) string {
	if s.Graph == nil {
		return ""
	}
	rest := strings.TrimPrefix(uri, "down://")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	kind, name := parts[0], parts[1]
	switch kind {
	case "tag", "mention", "entity":
		if loc := s.firstEntityLocation(name); loc != nil {
			return string(loc.URI)
		}
	case "task":
		for docURI, text := range s.Documents {
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if m := reTask.FindStringSubmatch(line); m != nil && strings.Contains(strings.ToLower(m[2]), strings.ToLower(name)) {
					return docURI + "#L" + intStr(i+1)
				}
			}
		}
	}
	_ = knowledge.KindTag
	return ""
}

func cleanURI(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}
