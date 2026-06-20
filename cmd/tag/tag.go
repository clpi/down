package tag

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp/handler"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

var tagRoot string

func scanTags(root string) ([]*knowledge.Entity, int) {
	home, _ := os.UserHomeDir()
	storePath := strings.TrimSuffix(home, "/") + "/.down/knowledge.json"
	state := &handler.State{
		Graph:     knowledge.NewFreshGraph(storePath),
		Documents: make(map[string]string),
	}
	files, _ := wsutil.WalkMarkdown(root, true)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		uri := "file://" + path
		state.Documents[uri] = string(data)
		knowledge.ExtractFromDocument(state.Graph, uri, string(data))
	}
	return state.Graph.EntitiesByKind(knowledge.KindTag), len(files)
}

var Tag = cobra.Command{
	Use:   "tag",
	Short: "List and search #tags in the workspace",
	Long:  "List tags from the knowledge graph (Notion-style tags).",
}

var tagList = cobra.Command{
	Use:   "list [query]",
	Short: "List tags with mention counts",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(args[0])
		}
		root := wsutil.ResolveRoot(tagRoot)
		tags, n := scanTags(root)
		if query != "" {
			filtered := tags[:0]
			for _, t := range tags {
				if strings.Contains(strings.ToLower(t.Name), query) {
					filtered = append(filtered, t)
				}
			}
			tags = filtered
		}
		if len(tags) == 0 {
			fmt.Printf("No tags found in %s (%d documents scanned).\n", root, n)
			return
		}
		sort.Slice(tags, func(i, j int) bool {
			if tags[i].Mentions != tags[j].Mentions {
				return tags[i].Mentions > tags[j].Mentions
			}
			return tags[i].Name < tags[j].Name
		})
		for _, t := range tags {
			fmt.Printf("#%-24s %3d mentions\n", t.Name, t.Mentions)
		}
		fmt.Printf("\n%d tag(s) across %d document(s).\n", len(tags), n)
	},
}

func init() {
	Tag.PersistentFlags().StringVar(&tagRoot, "root", "", "Workspace root")
	Tag.AddCommand(&tagList)
	Tag.Run = tagList.Run
}
