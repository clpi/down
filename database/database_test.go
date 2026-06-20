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
