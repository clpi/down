package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/clpi/down/lsp/knowledge"
)

type Engine struct {
	provider    Provider
	graph       *knowledge.Graph
	mu          sync.RWMutex
	history     []Message
	maxHistory  int
}

func NewEngine(provider Provider, graph *knowledge.Graph) *Engine {
	return &Engine{
		provider:   provider,
		graph:      graph,
		maxHistory: 20,
	}
}

func (e *Engine) ProviderName() string {
	return e.provider.Name()
}

func (e *Engine) ClearHistory() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = nil
}

func (e *Engine) appendHistory(role, content string) {
	e.history = append(e.history, Message{Role: role, Content: content})
	if len(e.history) > e.maxHistory {
		e.history = e.history[len(e.history)-e.maxHistory:]
	}
}

func (e *Engine) buildSystemPrompt(docURI string) string {
	var sb strings.Builder
	sb.WriteString("You are an AI writing assistant embedded in a markdown editor. ")
	sb.WriteString("You have deep knowledge of the user's workspace and documents.\n\n")

	sb.WriteString("## Knowledge Graph\n\n")
	sb.WriteString(e.graph.Summary())
	sb.WriteString("\n")

	entities := e.graph.AllEntities()
	if len(entities) > 0 {
		sb.WriteString("### Key Entities\n\n")
		grouped := make(map[knowledge.EntityKind][]*knowledge.Entity)
		for _, ent := range entities {
			grouped[ent.Kind] = append(grouped[ent.Kind], ent)
		}
		for kind, ents := range grouped {
			sb.WriteString("**" + string(kind) + "**: ")
			names := make([]string, 0, len(ents))
			for _, e := range ents {
				if len(names) >= 20 {
					names = append(names, fmt.Sprintf("...and %d more", len(ents)-20))
					break
				}
				names = append(names, e.Name)
			}
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if docURI != "" {
		docEntities := e.graph.EntitiesByDocument(docURI)
		if len(docEntities) > 0 {
			sb.WriteString("### Current Document Entities\n\n")
			for _, ent := range docEntities {
				rels := e.graph.RelationsFrom(ent.ID)
				sb.WriteString("- **" + ent.Name + "** (" + string(ent.Kind) + ")")
				if len(rels) > 0 {
					relStrs := make([]string, 0, len(rels))
					for _, r := range rels {
						if target, ok := e.graph.Entities[r.To]; ok {
							relStrs = append(relStrs, string(r.Kind)+" "+target.Name)
						}
					}
					if len(relStrs) > 0 {
						sb.WriteString(" → " + strings.Join(relStrs, ", "))
					}
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("Use this knowledge to provide contextually relevant completions and answers. ")
	sb.WriteString("Reference known entities, relationships, and patterns from the workspace. ")
	sb.WriteString("Keep responses concise and in markdown format.\n")

	return sb.String()
}

func (e *Engine) CompleteText(ctx context.Context, docURI string, precedingText string, line string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	systemPrompt := e.buildSystemPrompt(docURI)

	prompt := fmt.Sprintf(
		"The user is writing a markdown document. Complete the current line naturally.\n\n"+
			"Preceding context (last ~20 lines):\n```\n%s\n```\n\n"+
			"Current line so far: `%s`\n\n"+
			"Provide 3 possible completions for the current line, one per line. "+
			"Each completion should only contain the text to append (not the existing text). "+
			"Do not number them. Do not add explanations.",
		precedingText, line,
	)

	resp, err := e.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     []Message{{Role: "user", Content: prompt}},
		MaxTokens:    256,
		Temperature:  0.7,
	})
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(resp.Text), "\n")
	completions := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			completions = append(completions, l)
		}
	}
	return completions, nil
}

func (e *Engine) Query(ctx context.Context, docURI string, question string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	systemPrompt := e.buildSystemPrompt(docURI)

	e.appendHistory("user", question)

	msgs := make([]Message, len(e.history))
	copy(msgs, e.history)

	resp, err := e.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     msgs,
		MaxTokens:    2048,
		Temperature:  0.3,
	})
	if err != nil {
		e.history = e.history[:len(e.history)-1]
		return "", err
	}

	e.appendHistory("assistant", resp.Text)
	return resp.Text, nil
}

func (e *Engine) SuggestRelated(ctx context.Context, docURI string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	docEntities := e.graph.EntitiesByDocument(docURI)
	if len(docEntities) == 0 {
		return nil, nil
	}

	var entityNames []string
	for _, e := range docEntities {
		entityNames = append(entityNames, e.Name+" ("+string(e.Kind)+")")
	}

	systemPrompt := e.buildSystemPrompt(docURI)
	prompt := fmt.Sprintf(
		"Based on the entities in the current document [%s], "+
			"suggest 5 related topics or entities from the knowledge graph that could be relevant. "+
			"One per line, no explanations.",
		strings.Join(entityNames, ", "),
	)

	resp, err := e.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     []Message{{Role: "user", Content: prompt}},
		MaxTokens:    256,
		Temperature:  0.5,
	})
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(resp.Text), "\n")
	suggestions := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			suggestions = append(suggestions, l)
		}
	}
	return suggestions, nil
}

// TransformText applies an AI transformation to selected text.
// action is one of "expand", "summarize", "explain".
func (e *Engine) TransformText(ctx context.Context, docURI string, selectedText string, action string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	systemPrompt := e.buildSystemPrompt(docURI)

	var instruction string
	switch action {
	case "expand":
		instruction = "Expand the following text with more detail, examples, and elaboration. " +
			"Keep the same tone and style. Output only the expanded text in markdown, no preamble."
	case "summarize":
		instruction = "Summarize the following text concisely. " +
			"Capture the key points in a shorter form. Output only the summary in markdown, no preamble."
	case "explain":
		instruction = "Explain the following text clearly, as if to someone unfamiliar with the topic. " +
			"Define terms and give context. Output only the explanation in markdown, no preamble."
	default:
		instruction = "Rewrite the following text. Output only the rewritten text in markdown."
	}

	prompt := fmt.Sprintf("%s\n\n---\n\n%s", instruction, selectedText)

	resp, err := e.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     []Message{{Role: "user", Content: prompt}},
		MaxTokens:    2048,
		Temperature:  0.4,
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}


// InlineComplete generates ghost-text style inline completions.
// It produces continuations that flow naturally from the current line,
// taking into account both preceding and following context.
func (e *Engine) InlineComplete(ctx context.Context, docURI string, precedingText string, currentLine string, followingText string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	systemPrompt := e.buildSystemPrompt(docURI)
	systemPrompt += "\n\nYou are providing inline ghost-text completions. " +
		"Generate natural continuations that seamlessly extend the current line. " +
		"Keep completions concise (1-3 lines max). " +
		"Do not repeat the existing text. Only output the NEW text to append."

	prompt := fmt.Sprintf(
		"Context before cursor:\n```\n%s\n```\n\n"+
			"Current line (cursor at end): `%s`\n\n",
		precedingText, currentLine,
	)
	if followingText != "" {
		prompt += fmt.Sprintf("Context after cursor:\n```\n%s\n```\n\n", followingText)
	}
	prompt += "Provide 2 possible inline completions, separated by ---. " +
		"Each should be the text to INSERT at the cursor position. " +
		"Be concise. No explanations."

	resp, err := e.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     []Message{{Role: "user", Content: prompt}},
		MaxTokens:    200,
		Temperature:  0.6,
	})
	if err != nil {
		return nil, err
	}

	// Parse completions separated by ---
	parts := strings.Split(resp.Text, "---")
	completions := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && len(part) < 500 {
			completions = append(completions, part)
		}
	}
	if len(completions) > 3 {
		completions = completions[:3]
	}
	return completions, nil
}
