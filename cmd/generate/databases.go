package generate

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	dbpkg "github.com/clpi/down/database"
)

func renderDatabases(root string) string {
	dbs, err := dbpkg.ScanWorkspace(root)
	if err != nil {
		return "# Databases\n\nError scanning workspace: " + err.Error() + "\n"
	}
	ctx, _ := dbpkg.LoadWorkspaceContext(root)
	var b strings.Builder
	fmt.Fprintf(&b, "# Workspace Databases\n\n")
	fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	if len(dbs) == 0 {
		b.WriteString("No databases found (markdown files with type: database frontmatter).\n")
		return b.String()
	}
	fmt.Fprintf(&b, "**%d** database(s) in workspace.\n\n", len(dbs))
	b.WriteString("| Database | Rows | Columns | Path |\n")
	b.WriteString("| --- | ---: | ---: | --- |\n")
	for _, d := range dbs {
		rel, _ := filepath.Rel(root, d.Path)
		cols := len(d.ColumnNames())
		fmt.Fprintf(&b, "| [[%s]] | %d | %d | `%s` |\n", d.Title, len(d.Rows), cols, rel)
	}
	b.WriteString("\n## Schemas\n\n")
	for _, d := range dbs {
		rel, _ := filepath.Rel(root, d.Path)
		fmt.Fprintf(&b, "### %s\n\n", d.Title)
		fmt.Fprintf(&b, "`%s` · %d row(s)\n\n", rel, len(d.Rows))
		if len(d.Schema) > 0 {
			b.WriteString("| Column | Type | Options / Relation |\n")
			b.WriteString("| --- | --- | --- |\n")
			for _, col := range d.ColumnNames() {
				fd := d.Schema[col]
				extra := ""
				if len(fd.Options) > 0 {
					extra = strings.Join(fd.Options, ", ")
				}
				if fd.Database != "" {
					extra = "→ " + fd.Database
				}
				if fd.Type == "formula" && fd.Formula != "" {
					extra = "`" + fd.Formula + "`"
				}
				if fd.Type == "rollup" {
					extra = fd.Relation + " → " + fd.Target + " (" + fd.Aggregate + ")"
				}
				fmt.Fprintf(&b, "| %s | %s | %s |\n", col, fd.Type, extra)
			}
			b.WriteString("\n")
		}
		rows := d.Rows
		if ctx != nil {
			rows = dbpkg.ResolveComputed(d, rows, ctx)
		}
		if len(rows) > 0 {
			b.WriteString(dbpkg.RenderTable(d, rows))
			b.WriteString("\n")
		}
	}
	return b.String()
}
