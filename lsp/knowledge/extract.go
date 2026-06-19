package knowledge

import (
	"regexp"
	"strings"
)

var (
	reTag       = regexp.MustCompile(`(?:^|\s)#([a-zA-Z][a-zA-Z0-9_/-]*)`)
	reMention   = regexp.MustCompile(`(?:^|\s)@([a-zA-Z][a-zA-Z0-9_.-]*)`)
	reWikiLink  = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reMdLink    = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reTask      = regexp.MustCompile(`^[\s]*[-*]\s+\[([ xX])\]\s+(.+)`)
	reHeader    = regexp.MustCompile(`^(#{1,6})\s+(.+)`)
	reCodeBlock = regexp.MustCompile("^```")
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reRefDef    = regexp.MustCompile(`^\[([^\]]+)\]:\s+(\S+)`)
	reBlockquoteAttr = regexp.MustCompile(`^>\s*[-—]\s*(.+)$`)
	reFrontKey  = regexp.MustCompile(`^([a-zA-Z_]+):\s*(.+)`)
	reDate      = regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`)
)

func ExtractFromDocument(g *Graph, uri string, text string) {
	g.ClearDocument(uri)

	lines := strings.Split(text, "\n")
	inCode := false
	inFrontmatter := false
	var currentHeading *Entity
	docEntity := g.AddEntity(uri, KindDocument, Source{URI: uri, Line: 0})

	for i, line := range lines {
		src := Source{URI: uri, Line: i}

		if i == 0 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
				continue
			}
			extractFrontmatter(g, line, src, docEntity)
			continue
		}

		if reCodeBlock.MatchString(line) {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}

		if m := reHeader.FindStringSubmatch(line); m != nil {
			heading := strings.TrimSpace(m[2])
			currentHeading = g.AddEntity(heading, KindConcept, src)
			g.AddRelation(docEntity.ID, currentHeading.ID, RelPartOf, src)
			extractInline(g, heading, src, currentHeading)
			continue
		}

		if m := reRefDef.FindStringSubmatch(line); m != nil {
			label := m[1]
			target := m[2]
			linked := g.AddEntity(label, KindConcept, src)
			linked.Properties["url"] = target
			g.AddRelation(docEntity.ID, linked.ID, RelLinksTo, src)
			continue
		}

		if m := reBlockquoteAttr.FindStringSubmatch(line); m != nil {
			author := strings.TrimSpace(m[1])
			person := g.AddEntity(author, KindPerson, src)
			g.AddRelation(docEntity.ID, person.ID, RelMentions, src)
		}

		if m := reTask.FindStringSubmatch(line); m != nil {
			done := m[1] != " "
			taskText := strings.TrimSpace(m[2])
			task := g.AddEntity(taskText, KindAction, src)
			if done {
				task.Properties["status"] = "done"
			} else {
				task.Properties["status"] = "todo"
			}
			if currentHeading != nil {
				g.AddRelation(task.ID, currentHeading.ID, RelPartOf, src)
			}
			g.AddRelation(docEntity.ID, task.ID, RelMentions, src)
			extractInline(g, taskText, src, task)
			continue
		}

		extractInline(g, line, src, docEntity)
	}
}

func extractInline(g *Graph, text string, src Source, parent *Entity) {
	for _, m := range reTag.FindAllStringSubmatch(text, -1) {
		tag := g.AddEntity(m[1], KindTag, src)
		g.AddRelation(parent.ID, tag.ID, RelTaggedWith, src)
	}

	for _, m := range reMention.FindAllStringSubmatch(text, -1) {
		person := g.AddEntity(m[1], KindPerson, src)
		g.AddRelation(parent.ID, person.ID, RelMentions, src)
	}

	for _, m := range reWikiLink.FindAllStringSubmatch(text, -1) {
		parts := strings.SplitN(m[1], "|", 2)
		target := strings.TrimSpace(parts[0])
		linked := g.AddEntity(target, KindDocument, src)
		g.AddRelation(parent.ID, linked.ID, RelLinksTo, src)
	}

	for _, m := range reMdLink.FindAllStringSubmatch(text, -1) {
		linkText := m[1]
		linkTarget := m[2]
		if strings.HasPrefix(linkTarget, "http") {
			linked := g.AddEntity(linkText, KindConcept, src)
			linked.Properties["url"] = linkTarget
			g.AddRelation(parent.ID, linked.ID, RelLinksTo, src)
		} else {
			linked := g.AddEntity(linkTarget, KindDocument, src)
			g.AddRelation(parent.ID, linked.ID, RelLinksTo, src)
		}
	}

	for _, m := range reDate.FindAllStringSubmatch(text, -1) {
		date := g.AddEntity(m[1], KindDate, src)
		g.AddRelation(parent.ID, date.ID, RelScheduled, src)
	}

	for _, m := range reInlineCode.FindAllStringSubmatch(text, -1) {
		code := m[1]
		if len(code) >= 2 && len(code) <= 60 {
			codeEnt := g.AddEntity(code, KindCode, src)
			g.AddRelation(parent.ID, codeEnt.ID, RelMentions, src)
		}
	}
}

func extractFrontmatter(g *Graph, line string, src Source, docEntity *Entity) {
	m := reFrontKey.FindStringSubmatch(line)
	if m == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(m[1]))
	val := strings.TrimSpace(m[2])

	switch key {
	case "tags":
		for _, t := range strings.Split(val, ",") {
			t = strings.TrimSpace(strings.Trim(t, "[]\"' "))
			if t != "" {
				tag := g.AddEntity(t, KindTag, src)
				g.AddRelation(docEntity.ID, tag.ID, RelTaggedWith, src)
			}
		}
	case "author", "by", "assignee":
		person := g.AddEntity(val, KindPerson, src)
		g.AddRelation(docEntity.ID, person.ID, RelCreatedBy, src)
	case "project":
		proj := g.AddEntity(val, KindProject, src)
		g.AddRelation(docEntity.ID, proj.ID, RelPartOf, src)
	case "date", "due", "deadline":
		date := g.AddEntity(val, KindDate, src)
		g.AddRelation(docEntity.ID, date.ID, RelScheduled, src)
	case "title":
		docEntity.Name = val
	default:
		docEntity.Properties[key] = val
	}
}
