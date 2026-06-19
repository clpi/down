package semantic

import (
	"regexp"
	"sort"
	"strings"
)

var (
	reHeading        = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	reTag            = regexp.MustCompile(`(?:^|\s)#([A-Za-z][A-Za-z0-9_/-]*)`)
	reMention        = regexp.MustCompile(`(?:^|\s)@([A-Za-z][A-Za-z0-9_.-]*)`)
	reWikiLink       = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reMdLink         = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reTask           = regexp.MustCompile(`^(\s*[-*+] \[[ xX]\])`)
	reCodeSpan       = regexp.MustCompile("`([^`]+)`")
	reDate           = regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`)
	reFMKey          = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):\s`)
	reBold           = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reItalic         = regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*(?:[^*]|$)`)
	reBlockquote     = regexp.MustCompile(`^(>\s?.*)$`)
	reCodeFence      = regexp.MustCompile("^```")
	reStrikethrough  = regexp.MustCompile(`~~([^~]+)~~`)
	reHighlight      = regexp.MustCompile(`==([^=]+)==`)
	reFootnote       = regexp.MustCompile(`\[\^([^\]]+)\]`)
	reMathInline     = regexp.MustCompile(`\$([^$]+)\$`)
	reMathBlock      = regexp.MustCompile(`^\$\$`)
	reTableRow       = regexp.MustCompile(`^\|(.+)\|$`)
	reCallout        = regexp.MustCompile(`^>\s*\[!([a-zA-Z]+)\]`)
	reEmbed          = regexp.MustCompile(`!\[\[([^\]]+)\]\]`)
	reHorizontalRule = regexp.MustCompile(`^(---+|\*\*\*+|___+)\s*$`)
	reListMarker     = regexp.MustCompile(`^(\s*)([-*+]|\d+\.)\s`)
	wordRe           = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9_-]{1,})\b`)
)

// Tokenize scans a markdown document and returns sorted semantic tokens.
func Tokenize(text string) []Token {
	return TokenizeWithGraph(text, nil)
}

func TokenizeWithGraph(text string, knownEntities map[string]bool) []Token {
	lines := strings.Split(text, "\n")
	var tokens []Token

	inFrontmatter := false
	inCodeBlock := false
	inMathBlock := false
	fmStarted := false

	for lineIdx, line := range lines {
		// Track frontmatter boundaries
		if lineIdx == 0 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			fmStarted = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			continue
		}
		if inFrontmatter {
			if m := reFMKey.FindStringIndex(line); m != nil {
				keyEnd := strings.Index(line, ":")
				tokens = append(tokens, Token{
					Line:      lineIdx,
					StartChar: 0,
					Length:    keyEnd,
					Type:      TokFrontmatter,
					Modifiers: ModStatic,
				})
			}
			continue
		}

		// Math block $$...$$ 
		if reMathBlock.MatchString(strings.TrimSpace(line)) {
			if !inMathBlock {
				inMathBlock = true
				tokens = append(tokens, Token{
					Line:      lineIdx,
					StartChar: 0,
					Length:    len(line),
					Type:      TokMath,
				})
			} else {
				inMathBlock = false
				tokens = append(tokens, Token{
					Line:      lineIdx,
					StartChar: 0,
					Length:    len(line),
					Type:      TokMath,
				})
			}
			continue
		}
		if inMathBlock {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: 0,
				Length:    len(line),
				Type:      TokMath,
			})
			continue
		}

		// Track code fences
		if reCodeFence.MatchString(strings.TrimSpace(line)) {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Horizontal rules
		if reHorizontalRule.MatchString(strings.TrimSpace(line)) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: 0,
				Length:    len(line),
				Type:      TokHorizontalRule,
			})
			continue
		}

		// Callout blocks > [!type]
		if m := reCallout.FindStringIndex(line); m != nil {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokCallout,
				Modifiers: ModDeclaration,
			})
			// Don't continue — may have inline elements after callout header
		}

		// Headings
		if m := reHeading.FindStringSubmatchIndex(line); m != nil {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: 0,
				Length:    len(line),
				Type:      TokHeading,
				Modifiers: ModDeclaration,
			})
			// Still scan the heading text for inline elements below
		}

		// Blockquotes (if not a callout)
		if m := reBlockquote.FindStringIndex(line); m != nil {
			if !reCallout.MatchString(line) {
				tokens = append(tokens, Token{
					Line:      lineIdx,
					StartChar: m[0],
					Length:    m[1] - m[0],
					Type:      TokBlockquote,
				})
			}
		}

		// Table rows
		if m := reTableRow.FindStringIndex(line); m != nil {
			// Highlight the pipe delimiters
			for i, ch := range line {
				if ch == '|' {
					tokens = append(tokens, Token{
						Line:      lineIdx,
						StartChar: i,
						Length:    1,
						Type:      TokTableDelimiter,
					})
				}
			}
		}

		// List markers (only if not a task)
		if m := reListMarker.FindStringSubmatchIndex(line); m != nil {
			// Don't emit list marker for task items — task token covers it
			if !reTask.MatchString(line) {
				markerStart := m[4]
				markerEnd := m[5]
				tokens = append(tokens, Token{
					Line:      lineIdx,
					StartChar: markerStart,
					Length:    markerEnd - markerStart,
					Type:      TokListMarker,
				})
			}
		}

		// Tasks
		if m := reTask.FindStringIndex(line); m != nil {
			mod := 0
			if strings.Contains(line[m[0]:m[1]], "[x]") || strings.Contains(line[m[0]:m[1]], "[X]") {
				mod = ModDeprecated // completed task
			}
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokTask,
				Modifiers: mod,
			})
		}

		// Embeds ![[file]]
		for _, m := range reEmbed.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokEmbed,
				Modifiers: ModLink,
			})
		}

		// Tags (#tag)
		for _, m := range reTag.FindAllStringIndex(line, -1) {
			start := m[0]
			for start < m[1] && line[start] != '#' {
				start++
			}
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: start,
				Length:    m[1] - start,
				Type:      TokTag,
			})
		}

		// Mentions (@person)
		for _, m := range reMention.FindAllStringIndex(line, -1) {
			start := m[0]
			for start < m[1] && line[start] != '@' {
				start++
			}
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: start,
				Length:    m[1] - start,
				Type:      TokMention,
			})
		}

		// Wiki links [[target]]
		for _, m := range reWikiLink.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokWikiLink,
				Modifiers: ModLink,
			})
		}

		// Markdown links [text](url)
		for _, m := range reMdLink.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokMdLink,
				Modifiers: ModLink,
			})
		}

		// Footnotes [^ref]
		for _, m := range reFootnote.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokFootnote,
				Modifiers: ModLink,
			})
		}

		// Strikethrough ~~text~~
		for _, m := range reStrikethrough.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokStrikethrough,
				Modifiers: ModDeprecated,
			})
		}

		// Highlight ==text==
		for _, m := range reHighlight.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokHighlight,
			})
		}

		// Inline math $formula$
		for _, m := range reMathInline.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokMath,
			})
		}

		// Code spans `code`
		for _, m := range reCodeSpan.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokCodeSpan,
			})
		}

		// Dates (YYYY-MM-DD)
		for _, m := range reDate.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokDate,
				Modifiers: ModAsync,
			})
		}

		// Bold **text**
		for _, m := range reBold.FindAllStringIndex(line, -1) {
			tokens = append(tokens, Token{
				Line:      lineIdx,
				StartChar: m[0],
				Length:    m[1] - m[0],
				Type:      TokBold,
			})
		}

		// Italic *text* (not inside bold)
		for _, m := range reItalic.FindAllStringIndex(line, -1) {
			start := m[0]
			for start < m[1] && line[start] != '*' {
				start++
			}
			end := m[1]
			for end > start && line[end-1] != '*' {
				end--
			}
			if start < end {
				tokens = append(tokens, Token{
					Line:      lineIdx,
					StartChar: start,
					Length:    end - start,
					Type:      TokItalic,
				})
			}
		}
	}

	// Suppress the unused variable
	_ = fmStarted

	// Sort by line, then by start character
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].Line != tokens[j].Line {
			return tokens[i].Line < tokens[j].Line
		}
		return tokens[i].StartChar < tokens[j].StartChar
	})

	// Add knowledge graph entity tokens
	if knownEntities != nil && len(knownEntities) > 0 {
		for lineIdx, line := range lines {
			// Split into words and check against known entities
			wordRe.FindAllStringIndex(line, -1)
			words := wordRe.FindAllStringSubmatchIndex(line, -1)
			for _, w := range words {
				if len(w) >= 4 {
					word := line[w[2]:w[3]]
					if knownEntities[word] || knownEntities[strings.ToLower(word)] {
						tokens = append(tokens, Token{
							Line:      lineIdx,
							StartChar: w[2],
							Length:    w[3] - w[2],
							Type:      TokEntity,
							Modifiers: ModDeclaration,
						})
					}
				}
			}
		}
	}

	// Remove overlapping tokens (keep the first/more specific one)
	tokens = deduplicateTokens(tokens)

	return tokens
}

// deduplicateTokens removes overlapping tokens, keeping the first one found at each position.
func deduplicateTokens(tokens []Token) []Token {
	if len(tokens) <= 1 {
		return tokens
	}

	result := make([]Token, 0, len(tokens))
	result = append(result, tokens[0])

	for i := 1; i < len(tokens); i++ {
		prev := result[len(result)-1]
		curr := tokens[i]

		// Skip if this token overlaps with the previous one on the same line
		if curr.Line == prev.Line {
			prevEnd := prev.StartChar + prev.Length
			if curr.StartChar < prevEnd {
				continue // overlapping, skip
			}
		}
		result = append(result, curr)
	}
	return result
}
