package generate

import (
	"strings"
	"testing"

	"github.com/clpi/down/lsp/handler"
	"github.com/clpi/down/lsp/knowledge"
)

func TestRenderTasks(t *testing.T) {
	state := &handler.State{
		Graph: knowledge.NewFreshGraph(""),
		Documents: map[string]string{
			"file:///note/a.md": "# Alpha\n\n- [ ] First task\n- [x] Done task\n",
		},
	}
	knowledge.ExtractFromDocument(state.Graph, "file:///note/a.md", state.Documents["file:///note/a.md"])

	out := renderTasks("/workspace", state)
	if !strings.Contains(out, "First task") || !strings.Contains(out, "Done task") {
		t.Fatalf("expected tasks in output: %s", out)
	}
	if !strings.Contains(out, "**1 open**") {
		t.Fatalf("expected open count: %s", out)
	}
}

func TestRenderOrphans(t *testing.T) {
	state := &handler.State{
		Graph: knowledge.NewFreshGraph(""),
		Documents: map[string]string{
			"file:///orphan.md": "# Lonely\n\nNo links here.\n",
		},
	}
	knowledge.ExtractFromDocument(state.Graph, "file:///orphan.md", state.Documents["file:///orphan.md"])

	out := renderOrphans("/workspace", state)
	if !strings.Contains(out, "orphan.md") {
		t.Fatalf("expected orphan document: %s", out)
	}
}

func TestRenderStats(t *testing.T) {
	state := &handler.State{
		Graph: knowledge.NewFreshGraph(""),
		Documents: map[string]string{
			"file:///a.md": "# A\n\n#tag and @bob\n",
		},
	}
	knowledge.ExtractFromDocument(state.Graph, "file:///a.md", state.Documents["file:///a.md"])

	out := renderStats("/workspace", state, 1)
	if !strings.Contains(out, "Workspace Statistics") || !strings.Contains(out, "| Documents | 1 |") {
		t.Fatalf("unexpected stats output: %s", out)
	}
}
