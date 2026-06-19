package knowledge

import (
	"testing"
)

func TestDiagnoseBrokenWikiLink(t *testing.T) {
	g := NewGraph("")
	doc := "See [[Nonexistent Document]] for details"
	ExtractFromDocument(g, "file:///test.md", doc)

	diags := DiagnoseDocument(g, "file:///test.md", doc)
	found := false
	for _, d := range diags {
		if d.Severity == SeverityWarning && d.Line == 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for unresolved wiki link")
	}
}

func TestDiagnoseKnownWikiLink(t *testing.T) {
	g := NewGraph("")
	doc1 := "# Meeting Notes\nSome content"
	ExtractFromDocument(g, "file:///meeting-notes.md", doc1)

	doc2 := "See [[Meeting Notes]] for details"
	ExtractFromDocument(g, "file:///test.md", doc2)

	diags := DiagnoseDocument(g, "file:///test.md", doc2)
	for _, d := range diags {
		if d.Message == "Unresolved wiki link: Meeting Notes" {
			t.Error("wiki link to known entity should not produce warning")
		}
	}
}

func TestDiagnoseSkipsCodeBlocks(t *testing.T) {
	g := NewGraph("")
	doc := "Normal text\n```\n[[Not a real link]]\n```\nMore text"

	diags := DiagnoseDocument(g, "file:///test.md", doc)
	for _, d := range diags {
		if d.Line == 2 {
			t.Error("should not diagnose inside code blocks")
		}
	}
}
