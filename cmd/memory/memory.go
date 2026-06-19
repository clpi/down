package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type MemoryEntry struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Tags      []string          `json:"tags,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func dataDir() string {
	d := os.Getenv("XDG_DATA_HOME")
	if d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(d, "down", "memory")
}

func loadEntry(key string) (*MemoryEntry, error) {
	dir := dataDir()
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var e MemoryEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func saveEntry(e *MemoryEntry) error {
	dir := dataDir()
	os.MkdirAll(dir, 0755)
	e.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, e.Key+".json"), data, 0644)
}

func listEntries() ([]MemoryEntry, error) {
	dir := dataDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var mems []MemoryEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		entry, err := loadEntry(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		mems = append(mems, *entry)
	}
	sort.Slice(mems, func(i, j int) bool { return mems[i].UpdatedAt.After(mems[j].UpdatedAt) })
	return mems, nil
}

func deleteEntry(key string) error {
	return os.Remove(filepath.Join(dataDir(), key+".json"))
}

func searchEntries(query string) ([]MemoryEntry, error) {
	entries, err := listEntries()
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(query)
	var results []MemoryEntry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Key), lower) ||
			strings.Contains(strings.ToLower(e.Value), lower) {
			results = append(results, e)
			continue
		}
		for _, t := range e.Tags {
			if strings.Contains(strings.ToLower(t), lower) {
				results = append(results, e)
				break
			}
		}
	}
	return results, nil
}

var Memory = cobra.Command{
	Use:     "memory <command>",
	Aliases: []string{"mem", "m"},
	Short:   "Manage persistent AI memory",
	Long:    "Store, recall, and manage persistent memory entries for AI context.",
	Run: func(cmd *cobra.Command, args []string) {
		entries, _ := listEntries()
		if len(entries) == 0 {
			fmt.Println("No memory entries. Use `down memory add` to create one.")
			return
		}
		fmt.Printf("Memory entries (%d):\n\n", len(entries))
		for _, e := range entries {
			fmt.Printf("  %s  %s", e.UpdatedAt.Format("2006-01-02"), e.Key)
			if len(e.Tags) > 0 {
				fmt.Printf(" [%s]", strings.Join(e.Tags, ", "))
			}
			fmt.Println()
		}
	},
}

var memoryAdd = cobra.Command{
	Use:   "add <key> <value>",
	Short: "Add or update a memory entry",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := strings.Join(args[1:], " ")
		tags, _ := cmd.Flags().GetStringArray("tag")

		entry, err := loadEntry(key)
		if err != nil {
			entry = &MemoryEntry{
				Key:       key,
				CreatedAt: time.Now(),
				Meta:      make(map[string]string),
			}
		}
		entry.Value = value
		if len(tags) > 0 {
			entry.Tags = tags
		}
		if err := saveEntry(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Memory saved: %s\n", key)
	},
}

var memoryList = cobra.Command{
	Use:    "list",
	Short:  "List all memory entries",
	Run:    func(cmd *cobra.Command, args []string) { Memory.Run(cmd, args) },
}

var memoryShow = cobra.Command{
	Use:   "show <key>",
	Short: "Show a memory entry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		entry, err := loadEntry(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Not found: %s\n", args[0])
			os.Exit(1)
		}
		fmt.Printf("# %s\n\n", entry.Key)
		if len(entry.Tags) > 0 {
			fmt.Printf("Tags: %s\n\n", strings.Join(entry.Tags, ", "))
		}
		fmt.Println(entry.Value)
		fmt.Printf("\n---\nUpdated: %s\n", entry.UpdatedAt.Format("2006-01-02 15:04"))
	},
}

var memorySearch = cobra.Command{
	Use:   "search <query>",
	Short: "Search memory entries",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		results, err := searchEntries(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(results) == 0 {
			fmt.Printf("No results for: %s\n", args[0])
			return
		}
		fmt.Printf("Results for \"%s\" (%d):\n\n", args[0], len(results))
		for _, e := range results {
			fmt.Printf("## %s\n\n", e.Key)
			if len(e.Tags) > 0 {
				fmt.Printf("Tags: %s\n\n", strings.Join(e.Tags, ", "))
			}
			preview := e.Value
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Println(preview)
			fmt.Println()
		}
	},
}

var memoryDelete = cobra.Command{
	Use:    "delete <key>",
	Short:  "Delete a memory entry",
	Args:   cobra.ExactArgs(1),
	Aliases: []string{"rm", "del"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := deleteEntry(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted: %s\n", args[0])
	},
}

func init() {
	memoryAdd.Flags().StringArrayP("tag", "t", nil, "Tags for the memory entry")
	Memory.AddCommand(&memoryAdd)
	Memory.AddCommand(&memoryList)
	Memory.AddCommand(&memoryShow)
	Memory.AddCommand(&memorySearch)
	Memory.AddCommand(&memoryDelete)
}
