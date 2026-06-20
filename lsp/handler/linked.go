package handler

import (
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	reLinkedWiki    = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reLinkedTag     = regexp.MustCompile(`(?:^|\s)(#[A-Za-z][A-Za-z0-9_/-]*)`)
	reLinkedMention = regexp.MustCompile(`(?:^|\s)(@[A-Za-z][A-Za-z0-9_.-]*)`)
	reLinkedMdText  = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

// LinkedEditing returns ranges that should be edited together.
// Supports wiki links, tags, mentions, and markdown link labels.
func (s *State) LinkedEditing(_ *glsp.Context, p *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	uri := string(p.TextDocument.URI)
	text, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(text, "\n")
	line := int(p.Position.Line)
	col := int(p.Position.Character)
	if line >= len(lines) {
		return nil, nil
	}

	current := lines[line]

	if ranges, pattern := linkedRangesForWiki(current, line, lines, col); len(ranges) >= 2 {
		return &protocol.LinkedEditingRanges{Ranges: ranges, WordPattern: &pattern}, nil
	}
	if ranges, pattern := linkedRangesForTag(current, line, lines, col); len(ranges) >= 2 {
		return &protocol.LinkedEditingRanges{Ranges: ranges, WordPattern: &pattern}, nil
	}
	if ranges, pattern := linkedRangesForMention(current, line, lines, col); len(ranges) >= 2 {
		return &protocol.LinkedEditingRanges{Ranges: ranges, WordPattern: &pattern}, nil
	}
	if ranges, pattern := linkedRangesForMdLabel(current, line, lines, col); len(ranges) >= 2 {
		return &protocol.LinkedEditingRanges{Ranges: ranges, WordPattern: &pattern}, nil
	}

	return nil, nil
}

func linkedRangesForWiki(current string, line int, lines []string, col int) ([]protocol.Range, string) {
	var targetName string
	var innerStart, innerEnd int
	for _, m := range reLinkedWiki.FindAllStringSubmatchIndex(current, -1) {
		innerStart, innerEnd = m[2], m[3]
		if col >= innerStart && col <= innerEnd {
			targetName = current[innerStart:innerEnd]
			break
		}
	}
	if targetName == "" {
		return nil, ""
	}

	lower := strings.ToLower(strings.SplitN(targetName, "|", 2)[0])
	var ranges []protocol.Range
	for lineIdx, l := range lines {
		for _, m := range reLinkedWiki.FindAllStringSubmatchIndex(l, -1) {
			inner := l[m[2]:m[3]]
			innerKey := strings.ToLower(strings.SplitN(inner, "|", 2)[0])
			if innerKey == lower {
				ranges = append(ranges, makeRange(lineIdx, m[2], m[3]))
			}
		}
	}
	return ranges, `[^\]|]+`
}

func linkedRangesForTag(current string, line int, lines []string, col int) ([]protocol.Range, string) {
	var tagName string
	for _, m := range reLinkedTag.FindAllStringSubmatchIndex(current, -1) {
		start, end := m[2], m[3]
		if col >= start && col <= end {
			tagName = current[start:end]
			break
		}
	}
	if tagName == "" {
		return nil, ""
	}
	lower := strings.ToLower(tagName)
	var ranges []protocol.Range
	for lineIdx, l := range lines {
		for _, m := range reLinkedTag.FindAllStringSubmatchIndex(l, -1) {
			tag := l[m[2]:m[3]]
			if strings.ToLower(tag) == lower {
				ranges = append(ranges, makeRange(lineIdx, m[2], m[3]))
			}
		}
	}
	return ranges, `#[A-Za-z][A-Za-z0-9_/-]*`
}

func linkedRangesForMention(current string, line int, lines []string, col int) ([]protocol.Range, string) {
	var mentionName string
	for _, m := range reLinkedMention.FindAllStringSubmatchIndex(current, -1) {
		start, end := m[2], m[3]
		if col >= start && col <= end {
			mentionName = current[start:end]
			break
		}
	}
	if mentionName == "" {
		return nil, ""
	}
	lower := strings.ToLower(mentionName)
	var ranges []protocol.Range
	for lineIdx, l := range lines {
		for _, m := range reLinkedMention.FindAllStringSubmatchIndex(l, -1) {
			mention := l[m[2]:m[3]]
			if strings.ToLower(mention) == lower {
				ranges = append(ranges, makeRange(lineIdx, m[2], m[3]))
			}
		}
	}
	return ranges, `@[A-Za-z][A-Za-z0-9_.-]*`
}

func linkedRangesForMdLabel(current string, line int, lines []string, col int) ([]protocol.Range, string) {
	var label string
	for _, m := range reLinkedMdText.FindAllStringSubmatchIndex(current, -1) {
		start, end := m[2], m[3]
		if col >= start && col <= end {
			label = current[start:end]
			break
		}
	}
	if label == "" {
		return nil, ""
	}
	lower := strings.ToLower(label)
	var ranges []protocol.Range
	for lineIdx, l := range lines {
		for _, m := range reLinkedMdText.FindAllStringSubmatchIndex(l, -1) {
			text := l[m[2]:m[3]]
			if strings.ToLower(text) == lower {
				ranges = append(ranges, makeRange(lineIdx, m[2], m[3]))
			}
		}
	}
	return ranges, `[^\]]+`
}

func makeRange(lineIdx, start, end int) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(lineIdx), Character: protocol.UInteger(start)},
		End:   protocol.Position{Line: protocol.UInteger(lineIdx), Character: protocol.UInteger(end)},
	}
}
