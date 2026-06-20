package database

import (
	"fmt"
	"strings"
)

func RenderTable(db *Database, rows []Row) string {
	cols := db.ColumnNames()
	if len(cols) == 0 {
		return "_(empty database)_\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", db.Title)
	fmt.Fprintf(&b, "_%d row(s) · %d column(s)_\n\n", len(rows), len(cols))
	b.WriteString("| ")
	b.WriteString(strings.Join(cols, " | "))
	b.WriteString(" |\n| ")
	seps := make([]string, len(cols))
	for i := range seps {
		seps[i] = "---"
	}
	b.WriteString(strings.Join(seps, " | "))
	b.WriteString(" |\n")
	for _, row := range rows {
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = strings.ReplaceAll(row[c], "|", "\\|")
		}
		fmt.Fprintf(&b, "| %s |\n", strings.Join(cells, " | "))
	}
	return b.String()
}

func RenderBoard(db *Database, rows []Row, groupBy string) string {
	if groupBy == "" {
		groupBy = "status"
	}
	groups := GroupRows(rows, groupBy)
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	stringsSort(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — Board\n\n", db.Title)
	fmt.Fprintf(&b, "Grouped by **%s**\n\n", groupBy)
	for _, g := range keys {
		fmt.Fprintf(&b, "## %s (%d)\n\n", g, len(groups[g]))
		titleCol := db.TitleColumn()
		for _, row := range groups[g] {
			title := row[titleCol]
			if title == "" {
				title = "Untitled"
			}
			fmt.Fprintf(&b, "- %s\n", title)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func RenderList(db *Database, rows []Row, groupBy string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — List\n\n", db.Title)
	titleCol := db.TitleColumn()
	if groupBy != "" {
		groups := GroupRows(rows, groupBy)
		keys := make([]string, 0, len(groups))
		for k := range groups {
			keys = append(keys, k)
		}
		stringsSort(keys)
		for _, g := range keys {
			fmt.Fprintf(&b, "## %s\n\n", g)
			for _, row := range groups[g] {
				fmt.Fprintf(&b, "- %s\n", row[titleCol])
			}
			b.WriteString("\n")
		}
		return b.String()
	}
	for _, row := range rows {
		fmt.Fprintf(&b, "- %s\n", row[titleCol])
	}
	return b.String()
}

func RenderCalendar(db *Database, rows []Row, dateField string) string {
	if dateField == "" {
		dateField = "due_date"
	}
	byDate := map[string][]Row{}
	for _, row := range rows {
		val := strings.TrimSpace(row[dateField])
		if len(val) >= 10 {
			byDate[val[:10]] = append(byDate[val[:10]], row)
		}
	}
	keys := make([]string, 0, len(byDate))
	for k := range byDate {
		keys = append(keys, k)
	}
	stringsSort(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — Calendar\n\n", db.Title)
	titleCol := db.TitleColumn()
	for _, d := range keys {
		fmt.Fprintf(&b, "## %s\n\n", d)
		for _, row := range byDate[d] {
			line := row[titleCol]
			if s := row["status"]; s != "" {
				line += " [" + s + "]"
			}
			fmt.Fprintf(&b, "- %s\n", line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func RenderCSV(db *Database, rows []Row) string {
	cols := db.ColumnNames()
	var b strings.Builder
	quoted := func(s string) string {
		return fmt.Sprintf("%q", strings.ReplaceAll(s, "\"", "\"\""))
	}
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = quoted(c)
	}
	b.WriteString(strings.Join(cells, ","))
	b.WriteString("\n")
	for _, row := range rows {
		for i, c := range cols {
			cells[i] = quoted(row[c])
		}
		b.WriteString(strings.Join(cells, ","))
		b.WriteString("\n")
	}
	return b.String()
}
