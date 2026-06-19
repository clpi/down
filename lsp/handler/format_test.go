package handler

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestFormatTrailingWhitespace(t *testing.T) {
	// Single trailing space should be removed
	input := "Hello \nWorld\t\n"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	lines := strings.Split(result, "\n")
	for i, l := range lines {
		if strings.TrimRight(l, " \t") != l {
			t.Errorf("line %d has trailing whitespace: %q", i, l)
		}
	}
}

func TestFormatPreservesLineBreak(t *testing.T) {
	// 2+ trailing spaces is an intentional markdown line break, normalized to 2
	input := "First line   \nSecond\n"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	if !strings.Contains(result, "First line  \n") {
		t.Error("should preserve markdown line break as exactly 2 trailing spaces")
	}
}

func TestFormatCollapseBlankLines(t *testing.T) {
	input := "First\n\n\n\n\nSecond\n"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	if strings.Contains(result, "\n\n\n") {
		t.Error("should collapse multiple blank lines to at most one")
	}
}

func TestFormatHeadingSpace(t *testing.T) {
	input := "##NoSpace\n###Also\n"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	if !strings.Contains(result, "## NoSpace") {
		t.Error("should add space after heading markers")
	}
	if !strings.Contains(result, "### Also") {
		t.Error("should add space after heading markers")
	}
}

func TestFormatPreservesCodeBlocks(t *testing.T) {
	input := "```\n##NotAHeading   \n  trailing   \n```\n"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	if !strings.Contains(result, "##NotAHeading   ") {
		t.Error("should not modify content inside code blocks")
	}
}

func TestFormatPreservesFrontmatter(t *testing.T) {
	input := "---\ntitle: My Doc   \ntags: [a]\n---\n# Content\n"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	// Frontmatter should be preserved as-is
	if !strings.Contains(result, "title: My Doc   ") {
		t.Error("should not modify frontmatter content")
	}
}

func TestFormatEndsWithNewline(t *testing.T) {
	input := "No trailing newline"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	if !strings.HasSuffix(result, "\n") {
		t.Error("formatted document should end with newline")
	}
}

func TestFormatBlankBeforeHeading(t *testing.T) {
	input := "Some text\n## Heading\n"
	opts := protocol.FormattingOptions{}
	result := formatMarkdown(input, opts)
	if !strings.Contains(result, "Some text\n\n## Heading") {
		t.Errorf("should add blank line before heading, got: %q", result)
	}
}
