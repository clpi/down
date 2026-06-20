package database

import "testing"

const sampleDB = `---
title: Tasks
type: database
database:
  title:
    type: title
  status:
    type: select
    options:
      - Todo
      - Done
---

| title | status |
| --- | --- |
| First | Todo |
| Second | Done |
`

func TestParseDatabase(t *testing.T) {
	db := Parse(sampleDB)
	if db.Title != "Tasks" {
		t.Fatalf("title=%q", db.Title)
	}
	if len(db.Rows) != 2 {
		t.Fatalf("rows=%d", len(db.Rows))
	}
	if db.Rows[0]["title"] != "First" {
		t.Fatalf("row0=%v", db.Rows[0])
	}
}

func TestFilterRows(t *testing.T) {
	db := Parse(sampleDB)
	filtered := FilterRows(db.Rows, []Filter{{Property: "status", Operator: "eq", Value: "Done"}})
	if len(filtered) != 1 || filtered[0]["title"] != "Second" {
		t.Fatalf("filtered=%v", filtered)
	}
}


const rollupDB = `---
title: People
type: database
database:
  title:
    type: title
  tasks:
    type: relation
    database: Tasks
  open_tasks:
    type: rollup
    relation: tasks
    target: status
    aggregate: count
---

| title | tasks | open_tasks |
| --- | --- | --- |
| Alice | First | |
`

func TestResolveComputedFormula(t *testing.T) {
	db := Parse(sampleDB)
	db.Schema["score"] = FieldDef{Type: "formula", Formula: "{priority}"}
	rows := []Row{{"title": "A", "priority": "3"}}
	out := ResolveComputed(db, rows, nil)
	if out[0]["score"] != "3" {
		t.Fatalf("formula=%q", out[0]["score"])
	}
}

func TestResolveComputedRollupSameDB(t *testing.T) {
	db := Parse(sampleDB)
	db.Schema["done_count"] = FieldDef{Type: "rollup", Relation: "status", Target: "title", Aggregate: "count"}
	out := ResolveComputed(db, db.Rows, nil)
	if out[0]["done_count"] != "1" {
		t.Fatalf("rollup=%q", out[0]["done_count"])
	}
}
