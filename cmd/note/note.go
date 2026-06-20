package note

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var (
	noteRoot     string
	noteStrategy string
	noteTemplate string
	noteOpen     bool
)

func noteTime(sub string) time.Time {
	now := time.Now()
	switch sub {
	case "yesterday":
		return now.AddDate(0, 0, -1)
	case "tomorrow":
		return now.AddDate(0, 0, 1)
	case "week":
		year, week := now.ISOWeek()
		return time.Date(year, 1, 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, (week-1)*7)
	case "month":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "year":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		return now
	}
}

func openNote(t time.Time, label string) {
	root := wsutil.ResolveRoot(noteRoot)
	tmpl := noteTemplate
	if tmpl == "" {
		candidate := filepath.Join(root, "note", "day.md")
		if _, err := os.Stat(candidate); err == nil {
			tmpl = candidate
		}
	}
	path, err := wsutil.EnsureNoteAt(root, t, noteStrategy, tmpl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: %v\n", err)
		os.Exit(1)
	}
	if noteOpen {
		fmt.Printf("open %s\n", path)
	} else {
		fmt.Printf("%s\n", path)
	}
	if label != "" {
		fmt.Fprintf(os.Stderr, "Opened %s note: %s\n", label, path)
	}
}

var Note = cobra.Command{
	Use:     "note",
	Aliases: []string{"journal", "nt", "n"},
	Short:   "Daily notes and journal entries",
	Long:    "Open or create daily notes in the note/ folder (Notion-style journal).",
	Version: lsp.Version,
}

var noteToday = cobra.Command{
	Use:   "today",
	Short: "Open today's daily note",
	Run: func(cmd *cobra.Command, args []string) {
		openNote(time.Now(), "today")
	},
}

var noteYesterday = cobra.Command{
	Use:   "yesterday",
	Short: "Open yesterday's daily note",
	Run: func(cmd *cobra.Command, args []string) {
		openNote(noteTime("yesterday"), "yesterday")
	},
}

var noteTomorrow = cobra.Command{
	Use:   "tomorrow",
	Short: "Open tomorrow's daily note",
	Run: func(cmd *cobra.Command, args []string) {
		openNote(noteTime("tomorrow"), "tomorrow")
	},
}

var noteWeek = cobra.Command{
	Use:   "week",
	Short: "Open this week's note index",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(noteRoot)
		t := noteTime("week")
		_, wk := t.ISOWeek()
		path := filepath.Join(root, "note", fmt.Sprintf("%d-W%02d.md", t.Year(), wk))
		os.MkdirAll(filepath.Dir(path), 0755)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.WriteFile(path, []byte(fmt.Sprintf("# Week %02d %d\n\n", wk, t.Year())), 0644)
		}
		fmt.Println(path)
	},
}

var noteMonth = cobra.Command{
	Use:   "month",
	Short: "Open this month's note index",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(noteRoot)
		t := noteTime("month")
		path := filepath.Join(root, "note", t.Format("2006/01.md"))
		os.MkdirAll(filepath.Dir(path), 0755)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.WriteFile(path, []byte(fmt.Sprintf("# %s\n\n", t.Format("January 2006"))), 0644)
		}
		fmt.Println(path)
	},
}

var noteYear = cobra.Command{
	Use:   "year",
	Short: "Open this year's note index",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(noteRoot)
		t := noteTime("year")
		path := filepath.Join(root, "note", t.Format("2006.md"))
		os.MkdirAll(filepath.Dir(path), 0755)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.WriteFile(path, []byte(fmt.Sprintf("# %d\n\n", t.Year())), 0644)
		}
		fmt.Println(path)
	},
}

var notePath = cobra.Command{
	Use:   "path [date]",
	Short: "Print the path for a daily note (YYYY-MM-DD)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		t := time.Now()
		if len(args) > 0 {
			parsed, err := time.Parse("2006-01-02", args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid date: use YYYY-MM-DD\n")
				os.Exit(1)
			}
			t = parsed
		}
		fmt.Println(wsutil.NotePathFor(noteRoot, t, noteStrategy))
	},
}

func init() {
	Note.PersistentFlags().StringVar(&noteRoot, "root", "", "Workspace root (default: nearest .down/)")
	Note.PersistentFlags().StringVar(&noteStrategy, "strategy", "nested", "Note path strategy: nested or flat")
	Note.PersistentFlags().StringVar(&noteTemplate, "template", "", "Template file for new daily notes")
	Note.PersistentFlags().BoolVar(&noteOpen, "open", false, "Print 'open <path>' for editor integration")

	Note.AddCommand(&noteToday)
	Note.AddCommand(&noteYesterday)
	Note.AddCommand(&noteTomorrow)
	Note.AddCommand(&noteWeek)
	Note.AddCommand(&noteMonth)
	Note.AddCommand(&noteYear)
	Note.AddCommand(&notePath)

	Note.Run = func(cmd *cobra.Command, args []string) {
		openNote(time.Now(), "today")
	}
}
