package knowledge

import (
	"testing"
)

func TestExtractTags(t *testing.T) {
	g := NewGraph("")
	doc := "Some text with #golang and #rust tags"
	ExtractFromDocument(g, "file:///test.md", doc)

	tags := g.EntitiesByKind(KindTag)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestExtractMentions(t *testing.T) {
	g := NewGraph("")
	doc := "Assigned to @alice and reviewed by @bob"
	ExtractFromDocument(g, "file:///test.md", doc)

	people := g.EntitiesByKind(KindPerson)
	if len(people) != 2 {
		t.Errorf("expected 2 people, got %d", len(people))
	}
}

func TestExtractWikiLinks(t *testing.T) {
	g := NewGraph("")
	doc := "See [[Meeting Notes]] and [[Project Plan]]"
	ExtractFromDocument(g, "file:///test.md", doc)

	results := g.Search("Meeting Notes")
	if len(results) == 0 {
		t.Error("expected to find Meeting Notes entity")
	}
}

func TestExtractTasks(t *testing.T) {
	g := NewGraph("")
	doc := `# Todo
- [ ] Fix the build
- [x] Write tests
- [ ] Deploy to prod`
	ExtractFromDocument(g, "file:///test.md", doc)

	actions := g.EntitiesByKind(KindAction)
	if len(actions) != 3 {
		t.Errorf("expected 3 actions, got %d", len(actions))
	}

	var doneCount int
	for _, a := range actions {
		if a.Properties["status"] == "done" {
			doneCount++
		}
	}
	if doneCount != 1 {
		t.Errorf("expected 1 done task, got %d", doneCount)
	}
}

func TestExtractHeaders(t *testing.T) {
	g := NewGraph("")
	doc := `# Architecture
## Backend
### Database`
	ExtractFromDocument(g, "file:///test.md", doc)

	concepts := g.EntitiesByKind(KindConcept)
	if len(concepts) != 3 {
		t.Errorf("expected 3 concepts from headers, got %d", len(concepts))
	}
}

func TestExtractFrontmatter(t *testing.T) {
	g := NewGraph("")
	doc := `---
title: My Document
author: Chris
tags: go, lsp, ai
project: down.lsp
date: 2026-05-08
---
# Content here`
	ExtractFromDocument(g, "file:///test.md", doc)

	people := g.EntitiesByKind(KindPerson)
	if len(people) != 1 {
		t.Errorf("expected 1 person from author, got %d", len(people))
	}

	projects := g.EntitiesByKind(KindProject)
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}

	tags := g.EntitiesByKind(KindTag)
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
}

func TestExtractDates(t *testing.T) {
	g := NewGraph("")
	doc := "Meeting scheduled for 2026-05-15 and deadline is 2026-06-01"
	ExtractFromDocument(g, "file:///test.md", doc)

	dates := g.EntitiesByKind(KindDate)
	if len(dates) != 2 {
		t.Errorf("expected 2 dates, got %d", len(dates))
	}
}

func TestExtractSkipsCodeBlocks(t *testing.T) {
	g := NewGraph("")
	doc := "Text with #real-tag\n```\n#not-a-tag\n@not-a-mention\n```\nMore text with #another-tag"
	ExtractFromDocument(g, "file:///test.md", doc)

	tags := g.EntitiesByKind(KindTag)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags (skipping code block), got %d", len(tags))
	}
}

func TestExtractClearAndReExtract(t *testing.T) {
	g := NewGraph("")
	doc1 := "First version #old-tag"
	ExtractFromDocument(g, "file:///test.md", doc1)

	doc2 := "Updated version #new-tag"
	ExtractFromDocument(g, "file:///test.md", doc2)

	results := g.Search("old-tag")
	if len(results) != 0 {
		t.Error("old-tag should have been cleared on re-extract")
	}

	results = g.Search("new-tag")
	if len(results) != 1 {
		t.Error("new-tag should exist after re-extract")
	}
}

func TestExtractInlineCode(t *testing.T) {
	g := NewGraph("")
	doc := "Use `kubectl apply` and `docker build` to deploy"
	ExtractFromDocument(g, "file:///test.md", doc)

	codes := g.EntitiesByKind(KindCode)
	if len(codes) != 2 {
		t.Errorf("expected 2 code entities, got %d", len(codes))
	}
}

func TestExtractRefDefinition(t *testing.T) {
	g := NewGraph("")
	doc := "[Go]: https://golang.org\n[Rust]: https://rust-lang.org"
	ExtractFromDocument(g, "file:///test.md", doc)

	results := g.Search("Go")
	found := false
	for _, r := range results {
		if r.Kind == KindConcept && r.Properties["url"] == "https://golang.org" {
			found = true
		}
	}
	if !found {
		t.Error("expected Go concept with URL property")
	}
}

func TestExtractBlockquoteAttribution(t *testing.T) {
	g := NewGraph("")
	doc := "> The only way to do great work is to love what you do.\n> — Steve Jobs"
	ExtractFromDocument(g, "file:///test.md", doc)

	people := g.EntitiesByKind(KindPerson)
	if len(people) != 1 {
		t.Errorf("expected 1 person from attribution, got %d", len(people))
	}
	if len(people) > 0 && people[0].Name != "Steve Jobs" {
		t.Errorf("expected Steve Jobs, got %s", people[0].Name)
	}
}
