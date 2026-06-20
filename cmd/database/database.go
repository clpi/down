package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	db "github.com/clpi/down/database"
	"github.com/spf13/cobra"
)

var (
	dbRoot   string
	dbOutput string
	dbViewType string
	dbGroup  string
	dbDate   string
	dbWhere  []string
	dbSort   []string
	dbOpen   bool
	dbSchema string
	dbValues []string
)

var Database = cobra.Command{
	Use:     "database",
	Aliases: []string{"db", "dbs"},
	Short:   "Notion-style databases from markdown tables",
	Long: `Discover, query, and manage workspace databases stored as markdown tables
with YAML schema frontmatter (type: database). Supports table, board, list,
calendar views, filtering, sorting, and CSV export.`,
}

var dbList = cobra.Command{
	Use:   "list [query]",
	Short: "List databases in the workspace",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedDBRoot()
		dbs, err := db.ScanWorkspace(root)
		if err != nil {
			fail(err)
		}
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(args[0])
		}
		if len(dbs) == 0 {
			fmt.Printf("No databases found in %s\n", root)
			return
		}
		count := 0
		for _, d := range dbs {
			rel, _ := filepath.Rel(root, d.Path)
			if query != "" {
				blob := strings.ToLower(d.Title + " " + rel)
				if !strings.Contains(blob, query) {
					continue
				}
			}
			fmt.Printf("%-28s %3d rows  %s\n", d.Title, len(d.Rows), rel)
			count++
		}
		fmt.Printf("\n%d database(s) in %s\n", count, root)
	},
}

var dbShow = cobra.Command{
	Use:   "show <database>",
	Short: "Show database schema and row count",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedDBRoot()
		d, err := db.FindByName(root, args[0])
		if err != nil {
			fail(err)
		}
		rel, _ := filepath.Rel(root, d.Path)
		fmt.Printf("%s (%s)\n", d.Title, rel)
		fmt.Printf("Type: %s\n", d.Type)
		fmt.Printf("Rows: %d\n", len(d.Rows))
		fmt.Printf("Columns:\n")
		for _, col := range d.ColumnNames() {
			fd := d.Schema[col]
			line := fmt.Sprintf("  %-16s %s", col, fd.Type)
			if len(fd.Options) > 0 {
				line += " [" + strings.Join(fd.Options, ", ") + "]"
			}
			fmt.Println(line)
		}
	},
}

var dbQuery = cobra.Command{
	Use:   "query <database>",
	Short: "Query rows with filters and sorting",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedDBRoot()
		d, err := db.FindByName(root, args[0])
		if err != nil {
			fail(err)
		}
		rows := applyQuery(d)
		content := renderView(d, rows)
		if dbOutput != "" {
			if err := os.WriteFile(dbOutput, []byte(content), 0644); err != nil {
				fail(err)
			}
			fmt.Printf("Wrote %s\n", dbOutput)
			if dbOpen {
				fmt.Printf("open %s\n", dbOutput)
			}
			return
		}
		fmt.Print(content)
	},
}

var dbView = cobra.Command{
	Use:     "view <database> [table|board|list|calendar]",
	Short:   "Render a database view as markdown",
	Aliases: []string{"v"},
	Args:    cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			dbViewType = args[1]
		}
		dbQuery.Run(cmd, []string{args[0]})
	},
}

var dbCreate = cobra.Command{
	Use:   "create <name>",
	Short: "Create a new database markdown file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedDBRoot()
		path, err := db.CreateDatabase(root, args[0], dbSchema)
		if err != nil {
			fail(err)
		}
		fmt.Printf("Created: %s\n", path)
		if dbOpen {
			fmt.Printf("open %s\n", path)
		}
	},
}

var dbAddRow = cobra.Command{
	Use:     "add-row <database>",
	Short:   "Append a row to a database table",
	Aliases: []string{"add"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedDBRoot()
		d, err := db.FindByName(root, args[0])
		if err != nil {
			fail(err)
		}
		vals := parseKV(dbValues)
		if err := db.AddRow(d.Path, vals); err != nil {
			fail(err)
		}
		fmt.Printf("Added row to %s\n", d.Path)
	},
}

var dbExport = cobra.Command{
	Use:   "export <database>",
	Short: "Export database rows to CSV or markdown",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolvedDBRoot()
		d, err := db.FindByName(root, args[0])
		if err != nil {
			fail(err)
		}
		rows := applyQuery(d)
		format := strings.ToLower(dbViewType)
		if format == "" {
			format = "csv"
		}
		var content string
		switch format {
		case "md", "markdown", "table":
			content = db.RenderTable(d, rows)
		default:
			content = db.RenderCSV(d, rows)
		}
		if dbOutput == "" {
			fmt.Print(content)
			return
		}
		if err := os.WriteFile(dbOutput, []byte(content), 0644); err != nil {
			fail(err)
		}
		fmt.Printf("Exported %d row(s) to %s\n", len(rows), dbOutput)
	},
}


func resolvedDBRoot() string {
	if dbRoot != "" && dbRoot != "." {
		if abs, err := filepath.Abs(dbRoot); err == nil {
			return abs
		}
	}
	return wsutil.ResolveRoot(dbRoot)
}

func applyQuery(d *db.Database) []db.Row {
	rows := d.Rows
	filters := parseFilters(dbWhere)
	if len(filters) > 0 {
		rows = db.FilterRows(rows, filters)
	}
	sorts := parseSorts(dbSort)
	if len(sorts) > 0 {
		rows = db.SortRows(rows, sorts)
	}
	return rows
}

func renderView(d *db.Database, rows []db.Row) string {
	switch strings.ToLower(dbViewType) {
	case "board", "kanban":
		return db.RenderBoard(d, rows, dbGroup)
	case "list":
		return db.RenderList(d, rows, dbGroup)
	case "calendar", "cal":
		return db.RenderCalendar(d, rows, dbDate)
	default:
		return db.RenderTable(d, rows)
	}
}

func parseFilters(items []string) []db.Filter {
	var out []db.Filter
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		for _, sep := range []string{">=", "<=", "!=", "=", ">", "<", ":"} {
			if idx := strings.Index(item, sep); idx > 0 {
				prop := strings.TrimSpace(item[:idx])
				val := strings.TrimSpace(item[idx+len(sep):])
				op := "eq"
				switch sep {
				case ">":
					op = "gt"
				case "<":
					op = "lt"
				case ">=":
					op = "gte"
				case "<=":
					op = "lte"
				case "!=":
					op = "neq"
				}
				out = append(out, db.Filter{Property: prop, Operator: op, Value: val})
				break
			}
		}
	}
	return out
}

func parseSorts(items []string) []db.Sort {
	var out []db.Sort
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		dir := "ascending"
		if strings.HasPrefix(item, "-") {
			dir = "descending"
			item = strings.TrimPrefix(item, "-")
		}
		out = append(out, db.Sort{Property: item, Direction: dir})
	}
	return out
}

func parseKV(items []string) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		if idx := strings.Index(item, "="); idx > 0 {
			out[strings.TrimSpace(item[:idx])] = strings.TrimSpace(item[idx+1:])
		}
	}
	return out
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func init() {
	Database.PersistentFlags().StringVarP(&dbRoot, "root", "r", ".", "Workspace root")
	for _, c := range []*cobra.Command{&dbList, &dbShow, &dbQuery, &dbView, &dbCreate, &dbAddRow, &dbExport} {
		Database.AddCommand(c)
	}
	dbQuery.Flags().StringVarP(&dbOutput, "output", "o", "", "Write view to file")
	dbQuery.Flags().StringVar(&dbViewType, "view", "table", "View type: table, board, list, calendar")
	dbQuery.Flags().StringVar(&dbGroup, "group", "status", "Group-by column for board/list views")
	dbQuery.Flags().StringVar(&dbDate, "date", "due_date", "Date column for calendar view")
	dbQuery.Flags().StringArrayVar(&dbWhere, "where", nil, "Filter: column=value")
	dbQuery.Flags().StringArrayVar(&dbSort, "sort", nil, "Sort: column or -column")
	dbQuery.Flags().BoolVar(&dbOpen, "open", false, "Print path for editor to open")
	dbView.Flags().StringVarP(&dbOutput, "output", "o", "", "Write view to file")
	dbView.Flags().StringVar(&dbGroup, "group", "status", "Group-by column")
	dbView.Flags().StringVar(&dbDate, "date", "due_date", "Date column")
	dbView.Flags().StringArrayVar(&dbWhere, "where", nil, "Filter: column=value")
	dbView.Flags().StringArrayVar(&dbSort, "sort", nil, "Sort: column or -column")
	dbView.Flags().BoolVar(&dbOpen, "open", false, "Print path for editor to open")
	dbCreate.Flags().StringVar(&dbSchema, "schema", "tasks", "Schema preset: tasks, projects, blank")
	dbCreate.Flags().BoolVar(&dbOpen, "open", false, "Print path for editor to open")
	dbAddRow.Flags().StringArrayVar(&dbValues, "set", nil, "Column values: title=Foo")
	dbExport.Flags().StringVarP(&dbOutput, "output", "o", "", "Output file")
	dbExport.Flags().StringVar(&dbViewType, "format", "csv", "Export format: csv, md")
}
