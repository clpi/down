package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp/handler"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

var (
	genRoot     string
	genOutput   string
	genStrategy string
	genDays     int
	genLimit    int
	genOpen     bool
)

var reHeader = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
var reMermaidSafe = regexp.MustCompile(`[^a-z0-9_]`)

func scanWorkspace(root string) (*handler.State, []string, int) {
	home, _ := os.UserHomeDir()
	storePath := strings.TrimSuffix(home, "/") + "/.down/knowledge.json"
	state := &handler.State{
		Graph:     knowledge.NewFreshGraph(storePath),
		Documents: make(map[string]string),
	}
	files, _ := wsutil.WalkMarkdown(root, true)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		uri := "file://" + path
		state.Documents[uri] = string(data)
		knowledge.ExtractFromDocument(state.Graph, uri, string(data))
	}
	return state, files, len(files)
}

func resolvedRoot() string {
	return wsutil.ResolveRoot(genRoot)
}

func generatedDir(root string) string {
	downDir := wsutil.FindDownDir(root)
	if downDir == "" {
		downDir = filepath.Join(wsutil.ResolveRoot(root), ".down")
	}
	out := filepath.Join(downDir, "generated")
	_ = os.MkdirAll(out, 0755)
	return out
}

func outputPath(root, filename string) string {
	if genOutput != "" {
		return genOutput
	}
	return filepath.Join(generatedDir(root), filename)
}

func writeOutput(root, filename, content string) (string, error) {
	path := outputPath(root, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func relPath(root, abs string) string {
	if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return abs
}

func uriPath(uri string) string {
	p := strings.TrimPrefix(uri, "file://")
	p = strings.TrimPrefix(p, "file:")
	return p
}

func shortDoc(root, uri string) string {
	return relPath(root, uriPath(uri))
}

func finish(path string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated: %s\n", path)
	if genOpen {
		fmt.Printf("open %s\n", path)
	}
}

type dailyNote struct {
	Date    string
	Path    string
	Title   string
	Preview string
}

func parseDailyDate(path, noteDir string) string {
	base := filepath.Base(path)
	if len(base) == len("2006-01-02.md") && base[4] == '-' && base[7] == '-' {
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 0; i+2 < len(parts); i++ {
		if len(parts[i]) == 4 && len(parts[i+1]) == 2 {
			day := strings.TrimSuffix(parts[i+2], filepath.Ext(parts[i+2]))
			if len(day) == 2 {
				return parts[i] + "-" + parts[i+1] + "-" + day
			}
		}
	}
	return ""
}

func collectDailyNotes(root string) []dailyNote {
	noteDir := filepath.Join(wsutil.ResolveRoot(root), "note")
	strategy := strings.ToLower(strings.TrimSpace(genStrategy))
	if strategy == "" {
		strategy = "all"
	}
	var notes []dailyNote
	_ = filepath.Walk(noteDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !wsutil.MarkdownExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		rel, rerr := filepath.Rel(noteDir, path)
		if rerr != nil {
			return nil
		}
		switch strategy {
		case "flat":
			if strings.Contains(rel, string(filepath.Separator)) {
				return nil
			}
		case "nested":
			if !strings.Contains(rel, string(filepath.Separator)) {
				return nil
			}
		case "all", "auto", "both":
			// accept any dated layout
		default:
			// unknown strategy: accept any dated layout
		}
		date := parseDailyDate(path, noteDir)
		if date == "" {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		title := date
		preview := ""
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if m := reHeader.FindStringSubmatch(line); m != nil {
				title = strings.TrimSpace(m[2])
				break
			}
		}
		for _, line := range lines {
			t := strings.TrimSpace(line)
			if t != "" && !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "---") {
				preview = t
				if len(preview) > 80 {
					preview = preview[:80] + "..."
				}
				break
			}
		}
		notes = append(notes, dailyNote{
			Date:    date,
			Path:    relPath(root, path),
			Title:   title,
			Preview: preview,
		})
		return nil
	})
	sort.Slice(notes, func(i, j int) bool { return notes[i].Date > notes[j].Date })
	if genDays > 0 && len(notes) > genDays {
		notes = notes[:genDays]
	}
	return notes
}

func renderDaily(root string) string {
	notes := collectDailyNotes(root)
	var b strings.Builder
	fmt.Fprintf(&b, "# Daily Notes Index\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(notes) == 0 {
		b.WriteString("_No daily notes found under `note/`._\n")
		return b.String()
	}
	b.WriteString("| Date | Note | Preview |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, n := range notes {
		link := fmt.Sprintf("[[%s|%s]]", strings.TrimSuffix(filepath.Base(n.Path), filepath.Ext(n.Path)), n.Title)
		fmt.Fprintf(&b, "| %s | %s | %s |\n", n.Date, link, n.Preview)
	}
	fmt.Fprintf(&b, "\n_%d daily note(s)._\n", len(notes))
	return b.String()
}

type tocEntry struct {
	Path  string
	Level int
	Text  string
	Line  int
}

func collectTOC(root string, files []string) []tocEntry {
	var entries []tocEntry
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		rel := relPath(root, path)
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if m := reHeader.FindStringSubmatch(line); m != nil {
				entries = append(entries, tocEntry{
					Path:  rel,
					Level: len(m[1]),
					Text:  strings.TrimSpace(m[2]),
					Line:  i + 1,
				})
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Line < entries[j].Line
	})
	return entries
}

func renderTOC(root string, files []string) string {
	entries := collectTOC(root, files)
	var b strings.Builder
	fmt.Fprintf(&b, "# Workspace Table of Contents\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(entries) == 0 {
		b.WriteString("_No headings found._\n")
		return b.String()
	}
	cur := ""
	for _, e := range entries {
		if e.Path != cur {
			cur = e.Path
			fmt.Fprintf(&b, "\n## [%s](%s)\n\n", filepath.Base(cur), cur)
		}
		indent := strings.Repeat("  ", e.Level-1)
		fmt.Fprintf(&b, "%s- %s _(line %d)_\n", indent, e.Text, e.Line)
	}
	fmt.Fprintf(&b, "\n_%d heading(s) across workspace._\n", len(entries))
	return b.String()
}

func renderLinks(root string, state *handler.State) string {
	type linkRow struct {
		From string
		To   string
		Kind string
	}
	var rows []linkRow
	for _, rel := range state.Graph.Relations {
		if rel.Kind != knowledge.RelLinksTo && rel.Kind != knowledge.RelMentions && rel.Kind != knowledge.RelTaggedWith {
			continue
		}
		fromEnt, okFrom := state.Graph.Entities[rel.From]
		toEnt, okTo := state.Graph.Entities[rel.To]
		if !okFrom || !okTo {
			continue
		}
		fromDoc := fromEnt.Name
		if fromEnt.Kind == knowledge.KindDocument {
			fromDoc = shortDoc(root, fromEnt.Name)
		} else if len(fromEnt.Sources) > 0 {
			fromDoc = shortDoc(root, fromEnt.Sources[0].URI)
		}
		rows = append(rows, linkRow{
			From: fromDoc,
			To:   toEnt.Name,
			Kind: string(rel.Kind),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].From != rows[j].From {
			return rows[i].From < rows[j].From
		}
		return rows[i].To < rows[j].To
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# Compiled Links\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(rows) == 0 {
		b.WriteString("_No links found in knowledge graph._\n")
		return b.String()
	}
	b.WriteString("| Source | Target | Relation |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", r.From, r.To, r.Kind)
	}
	fmt.Fprintf(&b, "\n_%d link(s)._\n", len(rows))
	return b.String()
}

func renderTags(root string, state *handler.State) string {
	tags := state.Graph.EntitiesByKind(knowledge.KindTag)
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Mentions != tags[j].Mentions {
			return tags[i].Mentions > tags[j].Mentions
		}
		return tags[i].Name < tags[j].Name
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# Tag Index\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(tags) == 0 {
		b.WriteString("_No tags found._\n")
		return b.String()
	}
	for _, t := range tags {
		fmt.Fprintf(&b, "## #%s (%d)\n\n", t.Name, t.Mentions)
		docs := map[string]bool{}
		for _, rel := range state.Graph.Relations {
			if rel.To != t.ID && rel.From != t.ID {
				continue
			}
			otherID := rel.From
			if otherID == t.ID {
				otherID = rel.To
			}
			if ent, ok := state.Graph.Entities[otherID]; ok {
				for _, src := range ent.Sources {
					docs[shortDoc(root, src.URI)] = true
				}
			}
		}
		var docList []string
		for d := range docs {
			docList = append(docList, d)
		}
		sort.Strings(docList)
		for _, d := range docList {
			fmt.Fprintf(&b, "- [%s](%s)\n", filepath.Base(d), d)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "_%d tag(s)._\n", len(tags))
	return b.String()
}

func mermaidID(s string) string {
	s = strings.ToLower(s)
	s = reMermaidSafe.ReplaceAllString(s, "_")
	if s == "" {
		s = "node"
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "n_" + s
	}
	return s
}

func entityLabel(root string, ent *knowledge.Entity) string {
	if ent == nil {
		return ""
	}
	if ent.Kind == knowledge.KindDocument {
		return shortDoc(root, ent.Name)
	}
	if ent.Kind == knowledge.KindTag {
		return "#" + ent.Name
	}
	return ent.Name
}

func renderGraph(root string, state *handler.State) string {
	limit := genLimit
	if limit <= 0 {
		limit = 80
	}

	type edge struct {
		from, to, label string
	}
	seen := map[string]bool{}
	var edges []edge

	for _, rel := range state.Graph.Relations {
		fromEnt, okF := state.Graph.Entities[rel.From]
		toEnt, okT := state.Graph.Entities[rel.To]
		if !okF || !okT {
			continue
		}
		if fromEnt.Kind == knowledge.KindDocument || toEnt.Kind == knowledge.KindDocument {
			continue
		}
		if fromEnt.Kind == knowledge.KindCode && toEnt.Kind == knowledge.KindCode {
			continue
		}
		from := entityLabel(root, fromEnt)
		to := entityLabel(root, toEnt)
		if from == "" || to == "" || from == to {
			continue
		}
		key := from + "->" + to + ":" + string(rel.Kind)
		if seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, edge{
			from:  from,
			to:    to,
			label: string(rel.Kind),
		})
		if len(edges) >= limit {
			break
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Knowledge Graph\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "%s\n\n", state.Graph.Summary())
	b.WriteString("```mermaid\ngraph TD\n")
	ids := map[string]string{}
	for _, e := range edges {
		fid, ok := ids[e.from]
		if !ok {
			fid = mermaidID(e.from)
			ids[e.from] = fid
			fmt.Fprintf(&b, "  %s[\"%s\"]\n", fid, strings.ReplaceAll(e.from, `"`, `'`))
		}
		tid, ok := ids[e.to]
		if !ok {
			tid = mermaidID(e.to)
			ids[e.to] = tid
			fmt.Fprintf(&b, "  %s[\"%s\"]\n", tid, strings.ReplaceAll(e.to, `"`, `'`))
		}
		fmt.Fprintf(&b, "  %s -->|%s| %s\n", fid, e.label, tid)
	}
	b.WriteString("```\n")
	if len(edges) >= limit {
		fmt.Fprintf(&b, "\n_Showing first %d relations. Use --limit to adjust._\n", limit)
	}
	return b.String()
}

func docTitle(state *handler.State, uri string) string {
	if text, ok := state.Documents[uri]; ok {
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				return strings.TrimSpace(trimmed[2:])
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(uriPath(uri)), filepath.Ext(uriPath(uri)))
}

func renderTasks(root string, state *handler.State) string {
	result := state.ComputeTasks()
	open, done := 0, 0
	for _, t := range result.Tasks {
		if t.Completed {
			done++
		} else {
			open++
		}
	}

	byDoc := make(map[string][]handler.Task)
	order := make([]string, 0)
	for _, t := range result.Tasks {
		if _, ok := byDoc[t.URI]; !ok {
			order = append(order, t.URI)
		}
		byDoc[t.URI] = append(byDoc[t.URI], t)
	}
	sort.Slice(order, func(i, j int) bool {
		return shortDoc(root, order[i]) < shortDoc(root, order[j])
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# Workspace Tasks\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "**%d open** · **%d done** · %d total\n\n", open, done, result.Count)
	if result.Count == 0 {
		b.WriteString("_No checkbox tasks found._\n")
		return b.String()
	}
	for _, uri := range order {
		tasks := byDoc[uri]
		fmt.Fprintf(&b, "## %s\n\n", docTitle(state, uri))
		fmt.Fprintf(&b, "_%s_\n\n", shortDoc(root, uri))
		b.WriteString("| Status | Line | Task |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, t := range tasks {
			status := "open"
			if t.Completed {
				status = "done"
			}
			fmt.Fprintf(&b, "| %s | %d | %s |\n", status, t.Line+1, t.Text)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderMentions(root string, state *handler.State) string {
	people := state.Graph.EntitiesByKind(knowledge.KindPerson)
	sort.Slice(people, func(i, j int) bool {
		if people[i].Mentions != people[j].Mentions {
			return people[i].Mentions > people[j].Mentions
		}
		return people[i].Name < people[j].Name
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# Mentions Index\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(people) == 0 {
		b.WriteString("_No @mentions found._\n")
		return b.String()
	}
	b.WriteString("| Person | Mentions | Documents |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, p := range people {
		docs := map[string]bool{}
		for _, rel := range state.Graph.Relations {
			if rel.To != p.ID && rel.From != p.ID {
				continue
			}
			otherID := rel.From
			if otherID == p.ID {
				otherID = rel.To
			}
			if ent, ok := state.Graph.Entities[otherID]; ok {
				for _, src := range ent.Sources {
					docs[shortDoc(root, src.URI)] = true
				}
			}
		}
		var docList []string
		for d := range docs {
			docList = append(docList, d)
		}
		sort.Strings(docList)
		fmt.Fprintf(&b, "| @%s | %d | %s |\n", p.Name, p.Mentions, strings.Join(docList, ", "))
	}
	fmt.Fprintf(&b, "\n_%d person(s)._\n", len(people))
	return b.String()
}

type backlinkRow struct {
	URI   string
	Title string
	Path  string
	Count int
}

func collectBacklinkRows(root string, state *handler.State) []backlinkRow {
	rows := make([]backlinkRow, 0, len(state.Documents))
	for uri := range state.Documents {
		bl := state.ComputeBacklinks(uri)
		rows = append(rows, backlinkRow{
			URI:   uri,
			Title: bl.Title,
			Path:  shortDoc(root, uri),
			Count: bl.Count,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Path < rows[j].Path
	})
	return rows
}

func renderBacklinks(root string, state *handler.State) string {
	rows := collectBacklinkRows(root, state)
	linked := 0
	for _, r := range rows {
		if r.Count > 0 {
			linked++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Backlinks Index\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "**%d** documents with backlinks · **%d** total documents\n\n", linked, len(rows))
	if len(rows) == 0 {
		b.WriteString("_No documents indexed._\n")
		return b.String()
	}
	b.WriteString("| Document | Title | Backlinks |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, r := range rows {
		if r.Count == 0 {
			continue
		}
		fmt.Fprintf(&b, "| [%s](%s) | %s | %d |\n", filepath.Base(r.Path), r.Path, r.Title, r.Count)
	}

	limit := genLimit
	if limit <= 0 {
		limit = 20
	}
	shown := 0
	for _, r := range rows {
		if r.Count == 0 || shown >= limit {
			continue
		}
		bl := state.ComputeBacklinks(r.URI)
		fmt.Fprintf(&b, "\n## %s\n\n", r.Title)
		fmt.Fprintf(&b, "_%s — %d backlink(s)_\n\n", r.Path, r.Count)
		for _, link := range bl.Backlinks {
			fmt.Fprintf(&b, "- [%s](%s):%d — _%s_ (%s)\n",
				filepath.Base(shortDoc(root, link.SourceURI)),
				shortDoc(root, link.SourceURI),
				link.Line+1,
				link.Context,
				link.Kind,
			)
		}
		shown++
	}
	if shown < linked {
		fmt.Fprintf(&b, "\n_Showing detail for top %d documents. Use --limit to adjust._\n", shown)
	}
	return b.String()
}

func renderOrphans(root string, state *handler.State) string {
	rows := collectBacklinkRows(root, state)
	var orphans []backlinkRow
	for _, r := range rows {
		if r.Count == 0 {
			orphans = append(orphans, r)
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].Path < orphans[j].Path })

	var b strings.Builder
	fmt.Fprintf(&b, "# Orphan Documents\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Documents with **no inbound** wiki links, @mentions, tags, or graph relations.\n\n")
	if len(orphans) == 0 {
		b.WriteString("_No orphan documents — every file has at least one backlink._\n")
		return b.String()
	}
	for _, r := range orphans {
		fmt.Fprintf(&b, "- [%s](%s) — %s\n", r.Title, r.Path, r.Path)
	}
	fmt.Fprintf(&b, "\n_%d orphan document(s)._\n", len(orphans))
	return b.String()
}

func renderStats(root string, state *handler.State, fileCount int) string {
	tasks := state.ComputeTasks()
	openTasks := 0
	for _, t := range tasks.Tasks {
		if !t.Completed {
			openTasks++
		}
	}
	rows := collectBacklinkRows(root, state)
	orphans := 0
	for _, r := range rows {
		if r.Count == 0 {
			orphans++
		}
	}

	kindCounts := map[knowledge.EntityKind]int{}
	for _, ent := range state.Graph.Entities {
		kindCounts[ent.Kind]++
	}
	kinds := make([]knowledge.EntityKind, 0, len(kindCounts))
	for k := range kindCounts {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	var b strings.Builder
	fmt.Fprintf(&b, "# Workspace Statistics\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "| Metric | Value |\n")
	fmt.Fprintf(&b, "| --- | --- |\n")
	fmt.Fprintf(&b, "| Documents | %d |\n", fileCount)
	fmt.Fprintf(&b, "| Graph entities | %d |\n", len(state.Graph.Entities))
	fmt.Fprintf(&b, "| Graph relations | %d |\n", len(state.Graph.Relations))
	fmt.Fprintf(&b, "| Tags | %d |\n", len(state.Graph.EntitiesByKind(knowledge.KindTag)))
	fmt.Fprintf(&b, "| People | %d |\n", len(state.Graph.EntitiesByKind(knowledge.KindPerson)))
	fmt.Fprintf(&b, "| Tasks (open) | %d |\n", openTasks)
	fmt.Fprintf(&b, "| Tasks (total) | %d |\n", tasks.Count)
	fmt.Fprintf(&b, "| Documents with backlinks | %d |\n", len(rows)-orphans)
	fmt.Fprintf(&b, "| Orphan documents | %d |\n", orphans)
	fmt.Fprintf(&b, "| Daily notes | %d |\n", len(collectDailyNotes(root)))
	b.WriteString("\n## Entity breakdown\n\n")
	for _, k := range kinds {
		fmt.Fprintf(&b, "- **%s**: %d\n", k, kindCounts[k])
	}
	b.WriteString("\n")
	b.WriteString(state.Graph.Summary())
	b.WriteString("\n")
	return b.String()
}

func renderEntities(root string, state *handler.State) string {
	byKind := map[knowledge.EntityKind][]*knowledge.Entity{}
	for _, ent := range state.Graph.Entities {
		if ent.Kind == knowledge.KindDocument {
			continue
		}
		byKind[ent.Kind] = append(byKind[ent.Kind], ent)
	}
	kinds := make([]knowledge.EntityKind, 0, len(byKind))
	for k := range byKind {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	var b strings.Builder
	fmt.Fprintf(&b, "# Knowledge Entities\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(byKind) == 0 {
		b.WriteString("_No entities found._\n")
		return b.String()
	}
	total := 0
	for _, k := range kinds {
		ents := byKind[k]
		sort.Slice(ents, func(i, j int) bool {
			if ents[i].Mentions != ents[j].Mentions {
				return ents[i].Mentions > ents[j].Mentions
			}
			return ents[i].Name < ents[j].Name
		})
		fmt.Fprintf(&b, "## %s (%d)\n\n", k, len(ents))
		b.WriteString("| Name | Mentions | Sources |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, ent := range ents {
			docs := map[string]bool{}
			for _, src := range ent.Sources {
				docs[shortDoc(root, src.URI)] = true
			}
			var docList []string
			for d := range docs {
				docList = append(docList, d)
			}
			sort.Strings(docList)
			label := ent.Name
			if ent.Kind == knowledge.KindTag {
				label = "#" + ent.Name
			} else if ent.Kind == knowledge.KindPerson {
				label = "@" + ent.Name
			}
			fmt.Fprintf(&b, "| %s | %d | %s |\n", label, ent.Mentions, strings.Join(docList, ", "))
		}
		b.WriteString("\n")
		total += len(ents)
	}
	fmt.Fprintf(&b, "_%d entity(ies) across %d kind(s)._\n", total, len(kinds))
	return b.String()
}

func renderCalendar(root string) string {
	notes := collectDailyNotes(root)
	byMonth := map[string][]dailyNote{}
	for _, n := range notes {
		if len(n.Date) < 7 {
			continue
		}
		month := n.Date[:7]
		byMonth[month] = append(byMonth[month], n)
	}
	months := make([]string, 0, len(byMonth))
	for m := range byMonth {
		months = append(months, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))

	var b strings.Builder
	fmt.Fprintf(&b, "# Daily Notes Calendar\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(notes) == 0 {
		b.WriteString("_No daily notes found under `note/`._\n")
		return b.String()
	}
	for _, month := range months {
		fmt.Fprintf(&b, "## %s\n\n", month)
		b.WriteString("| Date | Note | Preview |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, n := range byMonth[month] {
			link := fmt.Sprintf("[[%s|%s]]", strings.TrimSuffix(filepath.Base(n.Path), filepath.Ext(n.Path)), n.Title)
			fmt.Fprintf(&b, "| %s | %s | %s |\n", n.Date, link, n.Preview)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "_%d daily note(s) across %d month(s)._\n", len(notes), len(months))
	return b.String()
}

func renderIndex(paths map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Generated Workspace Reports\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	b.WriteString("## Reports\n\n")
	names := []string{
		"stats.md",
		"daily-notes.md",
		"calendar.md",
		"toc.md",
		"tasks.md",
		"links.md",
		"tags.md",
		"mentions.md",
		"backlinks.md",
		"orphans.md",
		"entities.md",
		"graph.md",
	}
	for _, n := range names {
		if p, ok := paths[n]; ok {
			fmt.Fprintf(&b, "- [%s](%s)\n", n, filepath.Base(p))
		}
	}
	return b.String()
}

var Generate = cobra.Command{
	Use:     "generate [subcommand]",
	Aliases: []string{"gen", "g"},
	Short:   "Generate workspace reports and indexes",
	Long: `Generate markdown reports from the workspace knowledge graph and notes.

Writes output to .down/generated/ by default:
  daily-notes.md  — index of journal/daily notes
  toc.md          — workspace table of contents (all headings)
  links.md        — compiled wiki/markdown/tag links
  tags.md         — tag index with source documents
  graph.md        — Mermaid diagram of knowledge relations
  index.md        — hub page linking all reports (generate all)
  stats.md        — workspace statistics dashboard
  calendar.md     — daily notes grouped by month
  tasks.md        — checkbox tasks across documents
  mentions.md     — @mention index
  backlinks.md    — backlink counts and top referrers
  orphans.md      — documents with no inbound links
  entities.md     — knowledge graph entity listing`,
	Run: func(cmd *cobra.Command, args []string) {
		runAll()
	},
}

var genAll = cobra.Command{
	Use:   "all",
	Short: "Generate all workspace reports",
	Run: func(cmd *cobra.Command, args []string) {
		runAll()
	},
}

func runAll() {
	root := resolvedRoot()
	state, files, _ := scanWorkspace(root)
	paths := map[string]string{}

	p, err := writeOutput(root, "daily-notes.md", renderDaily(root))
	if err != nil {
		finish("", err)
	}
	paths["daily-notes.md"] = p

	p, err = writeOutput(root, "toc.md", renderTOC(root, files))
	if err != nil {
		finish("", err)
	}
	paths["toc.md"] = p

	p, err = writeOutput(root, "links.md", renderLinks(root, state))
	if err != nil {
		finish("", err)
	}
	paths["links.md"] = p

	p, err = writeOutput(root, "tags.md", renderTags(root, state))
	if err != nil {
		finish("", err)
	}
	paths["tags.md"] = p

	p, err = writeOutput(root, "stats.md", renderStats(root, state, len(files)))
	if err != nil {
		finish("", err)
	}
	paths["stats.md"] = p

	p, err = writeOutput(root, "calendar.md", renderCalendar(root))
	if err != nil {
		finish("", err)
	}
	paths["calendar.md"] = p

	p, err = writeOutput(root, "tasks.md", renderTasks(root, state))
	if err != nil {
		finish("", err)
	}
	paths["tasks.md"] = p

	p, err = writeOutput(root, "mentions.md", renderMentions(root, state))
	if err != nil {
		finish("", err)
	}
	paths["mentions.md"] = p

	p, err = writeOutput(root, "backlinks.md", renderBacklinks(root, state))
	if err != nil {
		finish("", err)
	}
	paths["backlinks.md"] = p

	p, err = writeOutput(root, "orphans.md", renderOrphans(root, state))
	if err != nil {
		finish("", err)
	}
	paths["orphans.md"] = p

	p, err = writeOutput(root, "entities.md", renderEntities(root, state))
	if err != nil {
		finish("", err)
	}
	paths["entities.md"] = p

	p, err = writeOutput(root, "graph.md", renderGraph(root, state))
	if err != nil {
		finish("", err)
	}
	paths["graph.md"] = p

	idx, err := writeOutput(root, "index.md", renderIndex(paths))
	if err != nil {
		finish("", err)
	}
	paths["index.md"] = idx

	fmt.Printf("Generated workspace reports in %s\n", generatedDir(root))
	for _, n := range []string{
		"stats.md", "daily-notes.md", "calendar.md", "toc.md", "tasks.md",
		"links.md", "tags.md", "mentions.md", "backlinks.md", "orphans.md",
		"entities.md", "graph.md", "index.md",
	} {
		if path, ok := paths[n]; ok {
			fmt.Printf("  %s\n", path)
		}
	}
	if genOpen {
		fmt.Printf("open %s\n", idx)
	}
}

var genDaily = cobra.Command{
	Use:   "daily",
	Short: "Generate daily notes index",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		path, err := writeOutput(root, "daily-notes.md", renderDaily(root))
		finish(path, err)
	},
}

var genTOC = cobra.Command{
	Use:     "toc",
	Aliases: []string{"table-of-contents"},
	Short:   "Generate workspace table of contents",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		_, files, _ := scanWorkspace(root)
		path, err := writeOutput(root, "toc.md", renderTOC(root, files))
		finish(path, err)
	},
}

var genLinks = cobra.Command{
	Use:   "links",
	Short: "Generate compiled links index",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "links.md", renderLinks(root, state))
		finish(path, err)
	},
}

var genTags = cobra.Command{
	Use:   "tags",
	Short: "Generate tag index",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "tags.md", renderTags(root, state))
		finish(path, err)
	},
}

var genGraph = cobra.Command{
	Use:     "graph",
	Aliases: []string{"diagram", "mermaid"},
	Short:   "Generate Mermaid knowledge graph diagram",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "graph.md", renderGraph(root, state))
		finish(path, err)
	},
}


var genTasks = cobra.Command{
	Use:     "tasks",
	Aliases: []string{"task", "todos"},
	Short:   "Generate workspace task report",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "tasks.md", renderTasks(root, state))
		finish(path, err)
	},
}

var genMentions = cobra.Command{
	Use:     "mentions",
	Aliases: []string{"mention", "@"},
	Short:   "Generate @mentions index",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "mentions.md", renderMentions(root, state))
		finish(path, err)
	},
}

var genBacklinks = cobra.Command{
	Use:     "backlinks",
	Aliases: []string{"bl", "backlink"},
	Short:   "Generate backlinks index",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "backlinks.md", renderBacklinks(root, state))
		finish(path, err)
	},
}

var genOrphans = cobra.Command{
	Use:     "orphans",
	Aliases: []string{"orphan", "unlinked"},
	Short:   "Generate orphan document report",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "orphans.md", renderOrphans(root, state))
		finish(path, err)
	},
}

var genStats = cobra.Command{
	Use:     "stats",
	Aliases: []string{"dashboard", "summary"},
	Short:   "Generate workspace statistics",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, n := scanWorkspace(root)
		path, err := writeOutput(root, "stats.md", renderStats(root, state, n))
		finish(path, err)
	},
}

var genEntities = cobra.Command{
	Use:     "entities",
	Aliases: []string{"entity", "knowledge"},
	Short:   "Generate knowledge entity listing",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		state, _, _ := scanWorkspace(root)
		path, err := writeOutput(root, "entities.md", renderEntities(root, state))
		finish(path, err)
	},
}

var genCalendar = cobra.Command{
	Use:     "calendar",
	Aliases: []string{"cal"},
	Short:   "Generate daily notes calendar by month",
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedRoot()
		path, err := writeOutput(root, "calendar.md", renderCalendar(root))
		finish(path, err)
	},
}
var genNotes = cobra.Command{
	Use:     "notes",
	Aliases: []string{"journal"},
	Short:   "Alias for daily notes index",
	Run: func(cmd *cobra.Command, args []string) {
		genDaily.Run(cmd, args)
	},
}

func init() {
	Generate.PersistentFlags().StringVar(&genRoot, "root", "", "Workspace root (default: nearest .down/ ancestor)")
	Generate.PersistentFlags().StringVarP(&genOutput, "output", "o", "", "Output file path (default: .down/generated/<name>.md)")
	Generate.PersistentFlags().StringVar(&genStrategy, "strategy", "all", "Daily note path strategy: all, nested, or flat")
	Generate.PersistentFlags().IntVar(&genDays, "days", 0, "Limit daily notes index to N most recent (0 = all)")
	Generate.PersistentFlags().IntVar(&genLimit, "limit", 80, "Max relations in graph diagram")
	Generate.PersistentFlags().BoolVar(&genOpen, "open", false, "Print 'open <path>' for editor integration")

	Generate.AddCommand(&genAll)
	Generate.AddCommand(&genDaily)
	Generate.AddCommand(&genTOC)
	Generate.AddCommand(&genLinks)
	Generate.AddCommand(&genTags)
	Generate.AddCommand(&genGraph)
	Generate.AddCommand(&genTasks)
	Generate.AddCommand(&genMentions)
	Generate.AddCommand(&genBacklinks)
	Generate.AddCommand(&genOrphans)
	Generate.AddCommand(&genStats)
	Generate.AddCommand(&genEntities)
	Generate.AddCommand(&genCalendar)
	Generate.AddCommand(&genNotes)
}
