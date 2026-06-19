package knowledge

import (
	"testing"
)

func TestAddEntity(t *testing.T) {
	g := NewGraph("")
	src := Source{URI: "file:///test.md", Line: 0}

	ent := g.AddEntity("Alice", KindPerson, src)
	if ent.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", ent.Name)
	}
	if ent.Kind != KindPerson {
		t.Errorf("expected kind person, got %s", ent.Kind)
	}
	if ent.Mentions != 1 {
		t.Errorf("expected 1 mention, got %d", ent.Mentions)
	}

	ent2 := g.AddEntity("Alice", KindPerson, Source{URI: "file:///other.md", Line: 5})
	if ent2.Mentions != 2 {
		t.Errorf("expected 2 mentions after re-add, got %d", ent2.Mentions)
	}
	if len(ent2.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(ent2.Sources))
	}
}

func TestSearch(t *testing.T) {
	g := NewGraph("")
	src := Source{URI: "file:///test.md", Line: 0}
	g.AddEntity("Kubernetes", KindConcept, src)
	g.AddEntity("Docker", KindConcept, src)
	g.AddEntity("kube-proxy", KindConcept, src)

	results := g.Search("kube")
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'kube', got %d", len(results))
	}
}

func TestRelations(t *testing.T) {
	g := NewGraph("")
	src := Source{URI: "file:///test.md", Line: 0}
	alice := g.AddEntity("Alice", KindPerson, src)
	project := g.AddEntity("ProjectX", KindProject, src)
	g.AddRelation(alice.ID, project.ID, RelPartOf, src)

	from := g.RelationsFrom(alice.ID)
	if len(from) != 1 {
		t.Errorf("expected 1 relation from Alice, got %d", len(from))
	}
	if from[0].Kind != RelPartOf {
		t.Errorf("expected part_of relation, got %s", from[0].Kind)
	}

	to := g.RelationsTo(project.ID)
	if len(to) != 1 {
		t.Errorf("expected 1 relation to ProjectX, got %d", len(to))
	}
}

func TestClearDocument(t *testing.T) {
	g := NewGraph("")
	src1 := Source{URI: "file:///a.md", Line: 0}
	src2 := Source{URI: "file:///b.md", Line: 0}
	g.AddEntity("SharedEntity", KindConcept, src1)
	g.AddEntity("SharedEntity", KindConcept, src2)
	g.AddEntity("OnlyInA", KindTag, src1)

	g.ClearDocument("file:///a.md")

	if _, ok := g.Entities[entityID(KindTag, "OnlyInA")]; ok {
		t.Error("OnlyInA should have been removed")
	}
	if ent, ok := g.Entities[entityID(KindConcept, "SharedEntity")]; !ok {
		t.Error("SharedEntity should still exist from b.md")
	} else if len(ent.Sources) != 1 {
		t.Errorf("SharedEntity should have 1 source, got %d", len(ent.Sources))
	}
}

func TestEntitiesByKind(t *testing.T) {
	g := NewGraph("")
	src := Source{URI: "file:///test.md", Line: 0}
	g.AddEntity("Alice", KindPerson, src)
	g.AddEntity("Bob", KindPerson, src)
	g.AddEntity("Go", KindConcept, src)

	people := g.EntitiesByKind(KindPerson)
	if len(people) != 2 {
		t.Errorf("expected 2 people, got %d", len(people))
	}
}
