package semantic

import (
	"testing"
)

func TestTokenizeHeading(t *testing.T) {
	tokens := Tokenize("# Hello World\n\nSome text")
	found := false
	for _, tok := range tokens {
		if tok.Type == TokHeading && tok.Line == 0 {
			found = true
			if tok.Modifiers&ModDeclaration == 0 {
				t.Error("heading should have declaration modifier")
			}
		}
	}
	if !found {
		t.Error("expected heading token on line 0")
	}
}

func TestTokenizeTags(t *testing.T) {
	tokens := Tokenize("Some text #golang and #rust")
	count := 0
	for _, tok := range tokens {
		if tok.Type == TokTag {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 tag tokens, got %d", count)
	}
}

func TestTokenizeMention(t *testing.T) {
	tokens := Tokenize("Talked to @alice about the project")
	found := false
	for _, tok := range tokens {
		if tok.Type == TokMention {
			found = true
		}
	}
	if !found {
		t.Error("expected mention token")
	}
}

func TestTokenizeWikiLink(t *testing.T) {
	tokens := Tokenize("See [[Meeting Notes]] for details")
	found := false
	for _, tok := range tokens {
		if tok.Type == TokWikiLink {
			found = true
			if tok.Modifiers&ModLink == 0 {
				t.Error("wiki link should have link modifier")
			}
		}
	}
	if !found {
		t.Error("expected wiki link token")
	}
}

func TestTokenizeDate(t *testing.T) {
	tokens := Tokenize("Due on 2024-03-15 for review")
	found := false
	for _, tok := range tokens {
		if tok.Type == TokDate {
			found = true
			if tok.Length != 10 {
				t.Errorf("date token length should be 10, got %d", tok.Length)
			}
		}
	}
	if !found {
		t.Error("expected date token")
	}
}

func TestTokenizeTask(t *testing.T) {
	tokens := Tokenize("- [ ] Open task\n- [x] Done task")
	open, done := false, false
	for _, tok := range tokens {
		if tok.Type == TokTask {
			if tok.Line == 0 {
				open = true
				if tok.Modifiers&ModDeprecated != 0 {
					t.Error("open task should not have deprecated modifier")
				}
			}
			if tok.Line == 1 {
				done = true
				if tok.Modifiers&ModDeprecated == 0 {
					t.Error("done task should have deprecated modifier")
				}
			}
		}
	}
	if !open {
		t.Error("expected open task token")
	}
	if !done {
		t.Error("expected done task token")
	}
}

func TestTokenizeCodeSpan(t *testing.T) {
	tokens := Tokenize("Use `fmt.Println` in Go")
	found := false
	for _, tok := range tokens {
		if tok.Type == TokCodeSpan {
			found = true
		}
	}
	if !found {
		t.Error("expected code span token")
	}
}

func TestTokenizeSkipsCodeBlocks(t *testing.T) {
	text := "# Heading\n```\n#not-a-tag\n@not-mention\n```\n#real-tag"
	tokens := Tokenize(text)
	tagCount := 0
	for _, tok := range tokens {
		if tok.Type == TokTag {
			tagCount++
		}
	}
	if tagCount != 1 {
		t.Errorf("expected 1 tag token (outside code block), got %d", tagCount)
	}
}

func TestTokenizeFrontmatter(t *testing.T) {
	text := "---\ntitle: My Doc\ntags: [a, b]\n---\n# Content"
	tokens := Tokenize(text)
	fmCount := 0
	for _, tok := range tokens {
		if tok.Type == TokFrontmatter {
			fmCount++
		}
	}
	if fmCount != 2 {
		t.Errorf("expected 2 frontmatter key tokens, got %d", fmCount)
	}
}

func TestEncodeDeltas(t *testing.T) {
	tokens := []Token{
		{Line: 0, StartChar: 0, Length: 5, Type: TokHeading, Modifiers: 0},
		{Line: 0, StartChar: 10, Length: 3, Type: TokTag, Modifiers: 0},
		{Line: 2, StartChar: 5, Length: 4, Type: TokMention, Modifiers: 0},
	}
	data := Encode(tokens)
	// 3 tokens × 5 ints = 15
	if len(data) != 15 {
		t.Fatalf("expected 15 ints, got %d", len(data))
	}
	// First token: deltaLine=0, deltaChar=0, len=5, type=heading(0), mod=0
	if data[0] != 0 || data[1] != 0 || data[2] != 5 {
		t.Errorf("first token wrong: %v", data[:5])
	}
	// Second token: deltaLine=0, deltaChar=10, len=3
	if data[5] != 0 || data[6] != 10 || data[7] != 3 {
		t.Errorf("second token wrong: %v", data[5:10])
	}
	// Third token: deltaLine=2, deltaChar=5 (new line, absolute), len=4
	if data[10] != 2 || data[11] != 5 || data[12] != 4 {
		t.Errorf("third token wrong: %v", data[10:15])
	}
}
