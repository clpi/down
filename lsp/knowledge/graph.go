package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type EntityKind string

const (
	KindPerson   EntityKind = "person"
	KindConcept  EntityKind = "concept"
	KindProject  EntityKind = "project"
	KindAction   EntityKind = "action"
	KindPlace    EntityKind = "place"
	KindDate     EntityKind = "date"
	KindTag      EntityKind = "tag"
	KindDocument EntityKind = "document"
	KindCode     EntityKind = "code"
)

type RelationKind string

const (
	RelMentions   RelationKind = "mentions"
	RelRelatesTo  RelationKind = "relates_to"
	RelDependsOn  RelationKind = "depends_on"
	RelCreatedBy  RelationKind = "created_by"
	RelTaggedWith RelationKind = "tagged_with"
	RelLinksTo    RelationKind = "links_to"
	RelPartOf     RelationKind = "part_of"
	RelAssignedTo RelationKind = "assigned_to"
	RelScheduled  RelationKind = "scheduled"
	RelBlocks     RelationKind = "blocks"
)

type Entity struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Kind       EntityKind        `json:"kind"`
	Properties map[string]string `json:"properties,omitempty"`
	Sources    []Source          `json:"sources"`
	FirstSeen  time.Time         `json:"first_seen"`
	LastSeen   time.Time         `json:"last_seen"`
	Mentions   int               `json:"mentions"`
}

type Source struct {
	URI  string `json:"uri"`
	Line int    `json:"line"`
}

type Relation struct {
	From       string            `json:"from"`
	To         string            `json:"to"`
	Kind       RelationKind      `json:"kind"`
	Properties map[string]string `json:"properties,omitempty"`
	Source     Source            `json:"source"`
	Created    time.Time         `json:"created"`
}

type Graph struct {
	mu        sync.RWMutex
	Entities  map[string]*Entity   `json:"entities"`
	Relations []*Relation          `json:"relations"`
	byKind    map[EntityKind][]*Entity
	byDoc     map[string][]*Entity
	storePath string
}

func NewGraph(storePath string) *Graph {
	g := &Graph{
		Entities:  make(map[string]*Entity),
		Relations: make([]*Relation, 0),
		byKind:    make(map[EntityKind][]*Entity),
		byDoc:     make(map[string][]*Entity),
		storePath: storePath,
	}
	g.load()
	return g
}

// NewFreshGraph creates a new empty graph bound to storePath without loading
// any previously persisted state. Use this for one-shot scans (e.g. CLI reads)
// that must reflect the current workspace rather than inherited mention counts.
func NewFreshGraph(storePath string) *Graph {
	return &Graph{
		Entities:  make(map[string]*Entity),
		Relations: make([]*Relation, 0),
		byKind:    make(map[EntityKind][]*Entity),
		byDoc:     make(map[string][]*Entity),
		storePath: storePath,
	}
}

func entityID(kind EntityKind, name string) string {
	return string(kind) + ":" + strings.ToLower(strings.TrimSpace(name))
}

func (g *Graph) AddEntity(name string, kind EntityKind, source Source) *Entity {
	g.mu.Lock()
	defer g.mu.Unlock()

	id := entityID(kind, name)
	now := time.Now()

	if ent, ok := g.Entities[id]; ok {
		ent.Mentions++
		ent.LastSeen = now
		found := false
		for _, s := range ent.Sources {
			if s.URI == source.URI && s.Line == source.Line {
				found = true
				break
			}
		}
		if !found {
			ent.Sources = append(ent.Sources, source)
		}
		return ent
	}

	ent := &Entity{
		ID:         id,
		Name:       name,
		Kind:       kind,
		Properties: make(map[string]string),
		Sources:    []Source{source},
		FirstSeen:  now,
		LastSeen:   now,
		Mentions:   1,
	}
	g.Entities[id] = ent
	g.byKind[kind] = append(g.byKind[kind], ent)
	g.byDoc[source.URI] = append(g.byDoc[source.URI], ent)
	return ent
}

func (g *Graph) AddRelation(fromID, toID string, kind RelationKind, source Source) *Relation {
	g.mu.Lock()
	defer g.mu.Unlock()

	rel := &Relation{
		From:       fromID,
		To:         toID,
		Kind:       kind,
		Properties: make(map[string]string),
		Source:     source,
		Created:    time.Now(),
	}
	g.Relations = append(g.Relations, rel)
	return rel
}

func (g *Graph) EntitiesByKind(kind EntityKind) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.byKind[kind]
}

func (g *Graph) EntitiesByDocument(uri string) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.byDoc[uri]
}

func (g *Graph) RelationsFrom(entityID string) []*Relation {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []*Relation
	for _, r := range g.Relations {
		if r.From == entityID {
			out = append(out, r)
		}
	}
	return out
}

func (g *Graph) RelationsTo(entityID string) []*Relation {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []*Relation
	for _, r := range g.Relations {
		if r.To == entityID {
			out = append(out, r)
		}
	}
	return out
}

func (g *Graph) Search(query string) []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*Entity
	for _, ent := range g.Entities {
		if strings.Contains(strings.ToLower(ent.Name), query) ||
			strings.Contains(ent.ID, query) {
			results = append(results, ent)
		}
	}
	return results
}

func (g *Graph) AllEntities() []*Entity {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Entity, 0, len(g.Entities))
	for _, e := range g.Entities {
		out = append(out, e)
	}
	return out
}

func (g *Graph) Summary() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	counts := make(map[EntityKind]int)
	for _, e := range g.Entities {
		counts[e.Kind]++
	}
	var sb strings.Builder
	sb.WriteString("Knowledge Graph:\n")
	for kind, count := range counts {
		sb.WriteString("  " + string(kind) + ": ")
		sb.WriteString(strings.Repeat("*", count))
		sb.WriteString(" (")
		sb.WriteString(intStr(count))
		sb.WriteString(")\n")
	}
	sb.WriteString("  relations: ")
	sb.WriteString(intStr(len(g.Relations)))
	sb.WriteString("\n")
	return sb.String()
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func (g *Graph) ClearDocument(uri string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for id, ent := range g.Entities {
		filtered := ent.Sources[:0]
		for _, s := range ent.Sources {
			if s.URI != uri {
				filtered = append(filtered, s)
			}
		}
		ent.Sources = filtered
		if len(ent.Sources) == 0 {
			delete(g.Entities, id)
		}
	}

	filtered := g.Relations[:0]
	for _, r := range g.Relations {
		if r.Source.URI != uri {
			filtered = append(filtered, r)
		}
	}
	g.Relations = filtered
	delete(g.byDoc, uri)

	g.rebuildIndexes()
}

func (g *Graph) rebuildIndexes() {
	g.byKind = make(map[EntityKind][]*Entity)
	g.byDoc = make(map[string][]*Entity)
	for _, ent := range g.Entities {
		g.byKind[ent.Kind] = append(g.byKind[ent.Kind], ent)
		for _, s := range ent.Sources {
			g.byDoc[s.URI] = append(g.byDoc[s.URI], ent)
		}
	}
}

func (g *Graph) Save() error {
	if g.storePath == "" {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	dir := filepath.Dir(g.storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(g.storePath, data, 0644)
}

func (g *Graph) load() {
	if g.storePath == "" {
		return
	}
	data, err := os.ReadFile(g.storePath)
	if err != nil {
		return
	}
	var loaded Graph
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}
	g.Entities = loaded.Entities
	g.Relations = loaded.Relations
	if g.Entities == nil {
		g.Entities = make(map[string]*Entity)
	}
	if g.Relations == nil {
		g.Relations = make([]*Relation, 0)
	}
	g.rebuildIndexes()
}
