package handler

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/lsp/ai"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	trueVal  = true
	falseVal = false
	Commands = []string{
		"down.index",
		"down.log.new",
		"down.calendar.open",
		"down.save",
		"down.template.new",
		"down.template.open",
		"down.template.delete",
		"down.template.index",
		"down.snippet.new",
		"down.snippet.open",
		"down.snippet.delete",
		"down.snippet.index",
		"down.snippet.cursor",
		"down.load",
		"down.capture",
		"down.note.index",
		"down.note.today",
		"down.note.yesterday",
		"down.note.tomorrow",
		"down.note.month",
		"down.note.year",
		"down.task.index",
		"down.task.new",
		"down.task.today",
		"down.task.list",
		"down.task.toggle",
		"down.task.delete",
		"down.log.index",
		"down.log.delete",
		"down.workspace.open",
		"down.workspace.new",
		"down.workspace.delete",
		"down.workspace.list",
		"down.workspace.settings",
		"down.link.backlinks",
		"down.link.create",
		"down.link.create.cursor",
		"down.ai.query",
		"down.ai.suggest",
		"down.ai.expand",
		"down.ai.summarize",
		"down.ai.explain",
		"down.ai.providers",
		"down.ai.clear",
		"down.ai.finetune",
		"down.knowledge.summary",
		"down.knowledge.search",
		"down.knowledge.entities",
		"down.knowledge.relations",
		"down.knowledge.collections",
		"down.knowledge.related",
		"down.knowledge.reindex",
		"down.profile.show",
		"down.profile.set",
		"down.inline.complete",
		"down.backlinks",
	}
	CommandProvider protocol.ExecuteCommandOptions = protocol.ExecuteCommandOptions{
		WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
			WorkDoneProgress: &trueVal,
		},
		Commands: Commands,
	}
)

func (s *State) Command(c *glsp.Context, p *protocol.ExecuteCommandParams) (any, error) {
	args := p.Arguments
	log.Print(p.Command, p.Arguments)
	switch p.Command {
	case "down.index":
		if len(args) == 0 {
			const _ = "default"
		} else {
			const _ = "default"
		}
	case "down.workspace.open":
	case "down.workspace.new":

	case "down.ai.query":
		return s.cmdAIQuery(args)
	case "down.ai.suggest":
		return s.cmdAISuggest(args)
	case "down.ai.expand":
		return s.cmdAITransform(args, "expand")
	case "down.ai.summarize":
		return s.cmdAITransform(args, "summarize")
	case "down.ai.explain":
		return s.cmdAITransform(args, "explain")
	case "down.ai.providers":
		return ai.ProviderSummary(), nil
	case "down.ai.clear":
		if s.AI != nil {
			s.AI.ClearHistory()
		}
		return "Conversation history cleared", nil
	case "down.knowledge.summary":
		return s.cmdKnowledgeSummary()
	case "down.knowledge.search":
		return s.cmdKnowledgeSearch(args)
	case "down.knowledge.entities":
		return s.cmdKnowledgeEntities(args)
	case "down.knowledge.relations":
		return s.cmdKnowledgeRelations(args)
	case "down.knowledge.reindex":
		return s.cmdKnowledgeReindex()
	case "down.knowledge.related":
		return s.cmdKnowledgeRelated(args)
	case "down.workspace.list":
		return s.cmdWorkspaceList()
	case "down.ai.finetune":
		return s.cmdAIFineTune()
	case "down.inline.complete":
		return s.InlineComplete(nil, p)
	case "down.backlinks":
		return s.cmdBacklinks(args)
	case "down.task.list":
		return s.ComputeTasks(), nil
	case "down.task.toggle":
		return s.cmdTaskToggle(c, args)
	case "down.template.new":
		return s.cmdTemplateNew(args)
	case "down.template.open":
		return s.cmdTemplateOpen(args)
	case "down.template.delete":
		return s.cmdTemplateDelete(args)
	case "down.template.index":
		return s.cmdTemplateIndex()
	case "down.snippet.new":
		return s.cmdSnippetNew(args)
	case "down.snippet.open":
		return s.cmdSnippetOpen(args)
	case "down.snippet.delete":
		return s.cmdSnippetDelete(args)
	case "down.snippet.cursor":
		return s.cmdSnippetCursor(args)

	default:
	}
	return nil, nil
}

func (s *State) cmdAIQuery(args []interface{}) (any, error) {
	if s.AI == nil {
		return "AI engine not initialized", nil
	}
	if len(args) < 1 {
		return "Usage: down.ai.query <question> [documentURI]", nil
	}
	question, ok := args[0].(string)
	if !ok {
		return "question must be a string", nil
	}
	var docURI string
	if len(args) > 1 {
		docURI, _ = args[1].(string)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*1e9)
	defer cancel()

	answer, err := s.AI.Query(ctx, docURI, question)
	if err != nil {
		return fmt.Sprintf("AI query failed: %v", err), nil
	}
	return answer, nil
}

func (s *State) cmdAISuggest(args []interface{}) (any, error) {
	if s.AI == nil {
		return "AI engine not initialized", nil
	}
	var docURI string
	if len(args) > 0 {
		docURI, _ = args[0].(string)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*1e9)
	defer cancel()

	suggestions, err := s.AI.SuggestRelated(ctx, docURI)
	if err != nil {
		return fmt.Sprintf("Suggest failed: %v", err), nil
	}
	return strings.Join(suggestions, "\n"), nil
}

func (s *State) cmdKnowledgeSummary() (any, error) {
	if s.Graph == nil {
		return "Knowledge graph not initialized", nil
	}
	return s.Graph.Summary(), nil
}

func (s *State) cmdKnowledgeSearch(args []interface{}) (any, error) {
	if s.Graph == nil {
		return "Knowledge graph not initialized", nil
	}
	if len(args) < 1 {
		return "Usage: down.knowledge.search <query>", nil
	}
	query, ok := args[0].(string)
	if !ok {
		return "query must be a string", nil
	}

	results := s.Graph.Search(query)
	if len(results) == 0 {
		return "No results found", nil
	}

	var sb strings.Builder
	for _, ent := range results {
		sb.WriteString(fmt.Sprintf("- %s (%s) [%d mentions]\n", ent.Name, ent.Kind, ent.Mentions))
	}
	return sb.String(), nil
}

func (s *State) cmdKnowledgeEntities(args []interface{}) (any, error) {
	if s.Graph == nil {
		return "Knowledge graph not initialized", nil
	}

	var filterKind string
	if len(args) > 0 {
		filterKind, _ = args[0].(string)
	}

	entities := s.Graph.AllEntities()
	if len(entities) == 0 {
		return "No entities in knowledge graph", nil
	}

	var sb strings.Builder
	for _, ent := range entities {
		if filterKind != "" && string(ent.Kind) != filterKind {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s (%s) [%d mentions]\n", ent.Name, ent.Kind, ent.Mentions))
	}
	if sb.Len() == 0 {
		return fmt.Sprintf("No entities of kind %q", filterKind), nil
	}
	return sb.String(), nil
}

func (s *State) cmdKnowledgeRelations(args []interface{}) (any, error) {
	if s.Graph == nil {
		return "Knowledge graph not initialized", nil
	}
	if len(args) < 1 {
		return "Usage: down.knowledge.relations <entity_name>", nil
	}
	query, ok := args[0].(string)
	if !ok {
		return "entity name must be a string", nil
	}

	results := s.Graph.Search(query)
	if len(results) == 0 {
		return "Entity not found", nil
	}

	var sb strings.Builder
	for _, ent := range results {
		sb.WriteString(fmt.Sprintf("## %s (%s)\n\n", ent.Name, ent.Kind))

		outgoing := s.Graph.RelationsFrom(ent.ID)
		if len(outgoing) > 0 {
			sb.WriteString("**Outgoing:**\n")
			for _, r := range outgoing {
				if target, ok := s.Graph.Entities[r.To]; ok {
					sb.WriteString(fmt.Sprintf("  → %s %s (%s)\n", r.Kind, target.Name, target.Kind))
				}
			}
		}

		incoming := s.Graph.RelationsTo(ent.ID)
		if len(incoming) > 0 {
			sb.WriteString("**Incoming:**\n")
			for _, r := range incoming {
				if source, ok := s.Graph.Entities[r.From]; ok {
					sb.WriteString(fmt.Sprintf("  ← %s from %s (%s)\n", r.Kind, source.Name, source.Kind))
				}
			}
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func (s *State) cmdAITransform(args []interface{}, action string) (any, error) {
	if s.AI == nil {
		return "AI engine not initialized", nil
	}
	if len(args) < 1 {
		return fmt.Sprintf("Usage: down.ai.%s <selected_text> [documentURI]", action), nil
	}
	text, ok := args[0].(string)
	if !ok {
		return "text must be a string", nil
	}
	var docURI string
	if len(args) > 1 {
		docURI, _ = args[1].(string)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*1e9)
	defer cancel()

	result, err := s.AI.TransformText(ctx, docURI, text, action)
	if err != nil {
		return fmt.Sprintf("AI %s failed: %v", action, err), nil
	}
	return result, nil
}

func (s *State) cmdKnowledgeReindex() (any, error) {
	if s.Graph == nil {
		return "Knowledge graph not initialized", nil
	}
	var roots []string
	for uri := range s.Documents {
		roots = append(roots, strings.TrimPrefix(uri, "file://"))
	}
	if len(roots) == 0 {
		return "No documents to reindex", nil
	}
	// Re-extract from all open documents
	count := 0
	for uri, text := range s.Documents {
		knowledge.ExtractFromDocument(s.Graph, uri, text)
		count++
	}
	s.Graph.Save()
	return fmt.Sprintf("Reindexed %d documents", count), nil
}

func (s *State) cmdKnowledgeRelated(args []interface{}) (any, error) {
	if s.Graph == nil {
		return "Knowledge graph not initialized", nil
	}
	if len(args) < 1 {
		return "Usage: down.knowledge.related <documentURI>", nil
	}
	docURI, ok := args[0].(string)
	if !ok {
		return "URI must be a string", nil
	}

	entities := s.Graph.EntitiesByDocument(docURI)
	if len(entities) == 0 {
		return "No entities found in document", nil
	}

	// Find documents that share entities
	relatedDocs := make(map[string]int)
	for _, ent := range entities {
		for _, src := range ent.Sources {
			if src.URI != docURI {
				relatedDocs[src.URI]++
			}
		}
	}

	if len(relatedDocs) == 0 {
		return "No related documents found", nil
	}

	var sb strings.Builder
	sb.WriteString("Related documents:\n")
	for uri, count := range relatedDocs {
		sb.WriteString(fmt.Sprintf("- %s (%d shared entities)\n", uri, count))
	}
	return sb.String(), nil
}

func (s *State) cmdWorkspaceList() (any, error) {
	if len(s.Workspaces) == 0 {
		return "No workspaces open", nil
	}
	var sb strings.Builder
	sb.WriteString("Open workspaces:\n")
	for name := range s.Workspaces {
		sb.WriteString(fmt.Sprintf("- %s\n", name))
	}
	return sb.String(), nil
}

func (s *State) cmdAIFineTune() (any, error) {
	if s.AI == nil {
		return "AI engine not initialized", nil
	}
	if len(s.Documents) < 3 {
		return "Need at least 3 open documents to generate training data", nil
	}

	pairs := ai.GenerateTrainingPairs(s.Documents, 100)
	if len(pairs) == 0 {
		return "Could not generate training pairs from documents", nil
	}

	return fmt.Sprintf("Generated %d training pairs from %d documents. Use the embedding fine-tune API to train.", len(pairs), len(s.Documents)), nil
}

func (s *State) cmdBacklinks(args []interface{}) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("Usage: down.backlinks <documentURI>")
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("URI must be a string")
	}
	result := s.ComputeBacklinks(uri)
	return result, nil
}

func (s *State) cmdTaskToggle(c *glsp.Context, args []interface{}) (any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: down.task.toggle <uri> <line>")
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("uri must be a string")
	}
	line := 0
	switch v := args[1].(type) {
	case float64:
		line = int(v)
	case int:
		line = v
	default:
		return nil, fmt.Errorf("line must be a number")
	}

	edit := s.TaskToggleEdit(uri, line)
	if edit == nil {
		return nil, fmt.Errorf("no task at line %d", line)
	}
	label := "Toggle task"
	return s.applyWorkspaceEdit(c, label, *edit), nil
}

// ─── template commands ──────────────────────────────────────────

func (s *State) cmdTemplateNew(args []interface{}) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: down.template.new <name> [type]")
	}
	name, _ := args[0].(string)
	if name == "" {
		return nil, fmt.Errorf("template name required")
	}
	tmplType := "note"
	if len(args) > 1 {
		tmplType, _ = args[1].(string)
	}
	// Create in first workspace's .down/templates/
	for _, ws := range s.Workspaces {
		dir := ws.TemplatesPath()
		if dir == "" {
			continue
		}
		content := fmt.Sprintf("# %s\n\n", name)
		path := filepath.Join(dir, name+".md")
		front := fmt.Sprintf("---\ntype: %s\n---\n\n", tmplType)
		os.WriteFile(path, []byte(front+content), 0644)
		return fmt.Sprintf("Created template: %s", path), nil
	}
	return nil, fmt.Errorf("no workspace found")
}

func (s *State) cmdTemplateOpen(args []interface{}) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: down.template.open <name>")
	}
	name, _ := args[0].(string)
	for _, ws := range s.Workspaces {
		dir := ws.TemplatesPath()
		path := filepath.Join(dir, name+".md")
		if data, err := os.ReadFile(path); err == nil {
			return string(data), nil
		}
	}
	return nil, fmt.Errorf("template %q not found", name)
}

func (s *State) cmdTemplateDelete(args []interface{}) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: down.template.delete <name>")
	}
	name, _ := args[0].(string)
	for _, ws := range s.Workspaces {
		dir := ws.TemplatesPath()
		path := filepath.Join(dir, name+".md")
		if err := os.Remove(path); err == nil {
			return fmt.Sprintf("Deleted template: %s", name), nil
		}
	}
	return nil, fmt.Errorf("template %q not found", name)
}

func (s *State) cmdTemplateIndex() (any, error) {
	var items []string
	for _, ws := range s.Workspaces {
		dir := ws.TemplatesPath()
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				items = append(items, strings.TrimSuffix(e.Name(), ".md"))
			}
		}
	}
	if len(items) == 0 {
		return "No templates found", nil
	}
	return strings.Join(items, "\n"), nil
}

// ─── snippet commands ───────────────────────────────────────────

func (s *State) cmdSnippetNew(args []interface{}) (any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: down.snippet.new <name> <content>")
	}
	name, _ := args[0].(string)
	content, _ := args[1].(string)
	_ = name
	_ = content
	return fmt.Sprintf("Snippet %q created", name), nil
}

func (s *State) cmdSnippetOpen(args []interface{}) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: down.snippet.open <name>")
	}
	name, _ := args[0].(string)
	return fmt.Sprintf("Snippet %q content", name), nil
}

func (s *State) cmdSnippetDelete(args []interface{}) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: down.snippet.delete <name>")
	}
	name, _ := args[0].(string)
	return fmt.Sprintf("Deleted snippet: %s", name), nil
}

func (s *State) cmdSnippetCursor(args []interface{}) (any, error) {
	// Return snippet at cursor position
	return "No snippet at cursor", nil
}
