package handler

import (
	"encoding/json"
	"strings"

	"github.com/clpi/down/lsp/knowledge"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// PrepareCallHierarchy implements textDocument/prepareCallHierarchy.
// Returns call-hierarchy items for headings, wiki-link targets, and tags.
func (s *State) PrepareCallHierarchy(_ *glsp.Context, p *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	if s.Graph == nil {
		return nil, nil
	}

	uri := string(p.TextDocument.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(p.Position.Line)
	if lineIdx >= len(lines) {
		return nil, nil
	}
	line := lines[lineIdx]
	col := int(p.Position.Character)
	if col >= len(line) {
		return nil, nil
	}

	// Heading
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		level := 0
		for _, ch := range trimmed {
			if ch == '#' {
				level++
			} else {
				break
			}
		}
		title := strings.TrimSpace(trimmed[level:])
		return []protocol.CallHierarchyItem{makeCallItem(title, protocol.SymbolKindKey, uri, lineIdx, 0, len(line), level)}, nil
	}

	// Wiki link
	for _, m := range reLinkWiki.FindAllStringSubmatchIndex(line, -1) {
		if col >= m[0] && col <= m[1] {
			parts := strings.SplitN(line[m[2]:m[3]], "|", 2)
			target := strings.TrimSpace(parts[0])
			return []protocol.CallHierarchyItem{makeCallItem(target, protocol.SymbolKindClass, uri, lineIdx, m[0], m[1], 0)}, nil
		}
	}

	// Tag or entity word
	word := wordAtPosition(line, col)
	if word != "" {
		results := s.Graph.Search(word)
		for _, ent := range results {
			if strings.EqualFold(ent.Name, word) {
				kind := protocol.SymbolKindString
				switch ent.Kind {
				case knowledge.KindPerson:
					kind = protocol.SymbolKindVariable
				case knowledge.KindProject:
					kind = protocol.SymbolKindPackage
				case knowledge.KindTag:
					kind = protocol.SymbolKindKey
				case knowledge.KindAction:
					kind = protocol.SymbolKindFunction
				case knowledge.KindConcept:
					kind = protocol.SymbolKindClass
				}
				return []protocol.CallHierarchyItem{makeCallItem(ent.Name, kind, uri, lineIdx, wordStart(line, col), wordEnd(line, col), 0)}, nil
			}
		}
	}

	return nil, nil
}

// CallHierarchyIncomingCalls returns documents that reference the given item.
func (s *State) CallHierarchyIncomingCalls(_ *glsp.Context, p *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	if s.Graph == nil {
		return nil, nil
	}

	item, err := decodeCallItem(p.Item.Data)
	if err != nil {
		item = p.Item
	}

	name := item.Name
	if name == "" {
		return nil, nil
	}

	results := s.Graph.Search(name)
	var calls []protocol.CallHierarchyIncomingCall
	for _, ent := range results {
		if !strings.EqualFold(ent.Name, name) {
			continue
		}
		for _, src := range ent.Sources {
			if src.URI == string(item.URI) && src.Line == int(item.Range.Start.Line) {
				continue
			}
			caller := makeCallItem(filepathName(src.URI), protocol.SymbolKindFile, src.URI, src.Line, 0, len(name), 0)
			calls = append(calls, protocol.CallHierarchyIncomingCall{
				From:       caller,
				FromRanges: []protocol.Range{entRange(src.Line, len(name))},
			})
		}
	}
	return calls, nil
}

// CallHierarchyOutgoingCalls returns links/relations from the given item to other items.
func (s *State) CallHierarchyOutgoingCalls(_ *glsp.Context, p *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	if s.Graph == nil {
		return nil, nil
	}

	item, err := decodeCallItem(p.Item.Data)
	if err != nil {
		item = p.Item
	}

	uri := string(item.URI)
	doc, ok := s.Documents[uri]
	if !ok {
		return nil, nil
	}

	lines := strings.Split(doc, "\n")
	lineIdx := int(item.Range.Start.Line)
	if lineIdx >= len(lines) {
		return nil, nil
	}
	line := lines[lineIdx]

	var calls []protocol.CallHierarchyOutgoingCall
	// Wiki links on the item's line are outgoing calls.
	for _, m := range reLinkWiki.FindAllStringSubmatchIndex(line, -1) {
		parts := strings.SplitN(line[m[2]:m[3]], "|", 2)
		target := strings.TrimSpace(parts[0])
		calls = append(calls, protocol.CallHierarchyOutgoingCall{
			To:         makeCallItem(target, protocol.SymbolKindClass, uri, lineIdx, m[0], m[1], 0),
			FromRanges: []protocol.Range{entRange(lineIdx, m[1]-m[0])},
		})
	}

	// Relations from knowledge graph for the item name.
	results := s.Graph.Search(item.Name)
	seen := make(map[string]bool)
	for _, ent := range results {
		if !strings.EqualFold(ent.Name, item.Name) {
			continue
		}
		for _, r := range s.Graph.RelationsFrom(ent.ID) {
			if target, ok := s.Graph.Entities[r.To]; ok {
				key := "to:" + target.Name
				if seen[key] {
					continue
				}
				seen[key] = true
				calls = append(calls, protocol.CallHierarchyOutgoingCall{
					To:         makeCallItem(target.Name, protocol.SymbolKindClass, uri, lineIdx, 0, len(target.Name), 0),
					FromRanges: []protocol.Range{entRange(lineIdx, len(target.Name))},
				})
			}
		}
	}

	return calls, nil
}

func makeCallItem(name string, kind protocol.SymbolKind, uri string, line, startCol, endCol, level int) protocol.CallHierarchyItem {
	detail := ""
	if level > 0 {
		detail = strings.Repeat("#", level) + " heading"
	}
	data := map[string]interface{}{
		"name":  name,
		"uri":   uri,
		"line":  line,
		"start": startCol,
		"end":   endCol,
		"level": level,
	}
	return protocol.CallHierarchyItem{
		Name:     name,
		Kind:     kind,
		Detail:   &detail,
		URI:      protocol.DocumentUri(uri),
		Range:    entRange(line, endCol-startCol),
		SelectionRange: entRange(line, endCol-startCol),
		Data:     data,
	}
}

func decodeCallItem(data any) (protocol.CallHierarchyItem, error) {
	var item protocol.CallHierarchyItem
	if data == nil {
		return item, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return item, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return item, err
	}
	name, _ := m["name"].(string)
	uri, _ := m["uri"].(string)
	line := 0
	if v, ok := m["line"].(float64); ok {
		line = int(v)
	}
	start := 0
	if v, ok := m["start"].(float64); ok {
		start = int(v)
	}
	end := 0
	if v, ok := m["end"].(float64); ok {
		end = int(v)
	}
	level := 0
	if v, ok := m["level"].(float64); ok {
		level = int(v)
	}
	item = makeCallItem(name, protocol.SymbolKindKey, uri, line, start, end, level)
	return item, nil
}

func filepathName(uri string) string {
	uri = strings.TrimPrefix(uri, "file://")
	uri = strings.TrimPrefix(uri, "file:")
	parts := strings.Split(uri, "/")
	if len(parts) == 0 {
		return uri
	}
	return parts[len(parts)-1]
}

func entRange(line, length int) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(line), Character: 0},
		End:   protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(length)},
	}
}

func wordStart(line string, col int) int {
	start := col
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	return start
}

func wordEnd(line string, col int) int {
	end := col
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	return end
}
