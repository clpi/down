package notion

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/pkg/notion"
	"github.com/spf13/cobra"
)

var (
	notionTitle   string
	notionParent  string
	notionDBID    string
	notionOutput  string
	notionFile    string
	notionContent string
	notionQuery   string
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func marshalProps() json.RawMessage {
	props := map[string]any{}
	if notionTitle != "" {
		props["title"] = map[string]any{
			"title": []map[string]any{
				{
					"type": "text",
					"text": map[string]any{"content": notionTitle},
				},
			},
		}
	}
	if len(props) == 0 {
		return json.RawMessage(`{"title":{"title":[{"type":"text","text":{"content":"New page"}}]}}`)
	}
	b, _ := json.Marshal(props)
	return b
}

func parentRef() map[string]any {
	id := notion.NormalizeID(notionParent)
	if notionDBID != "" {
		return map[string]any{"database_id": notion.NormalizeID(notionDBID)}
	}
	return map[string]any{"page_id": id}
}

func outputJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		die(err)
	}
	if notionOutput != "" {
		os.WriteFile(notionOutput, b, 0644)
		fmt.Printf("Wrote %s\n", notionOutput)
	} else {
		fmt.Println(string(b))
	}
}

func outputText(s string) {
	if notionOutput != "" {
		os.WriteFile(notionOutput, []byte(s), 0644)
		fmt.Printf("Wrote %s\n", notionOutput)
	} else {
		fmt.Println(s)
	}
}

// Notion is the top-level notion command.
var Notion = cobra.Command{
	Use:   "notion",
	Short: "Notion integration: search, pages, databases, blocks, files",
	Long: `Interact with the Notion API.

Subcommands:
  me                         Show the current bot user
  users                      List workspace users
  search <query>             Search pages and databases
  page get <id>              Get page metadata
  page markdown <id>         Get page as Markdown
  page create                Create a page from stdin or --content
  page update <id>           Update page properties
  page append <id>           Append blocks from stdin or --content
  db get <id>                Get database metadata
  db query <id>              Query database entries
  db create                  Create a database under --parent
  blocks <id>                List block children
  file upload <path>         Upload a file to Notion`,
}

var notionMe = cobra.Command{
	Use:   "me",
	Short: "Show the current bot user",
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		u, err := c.GetMe()
		if err != nil {
			die(err)
		}
		outputJSON(u)
	},
}

var notionUsers = cobra.Command{
	Use:   "users",
	Short: "List workspace users",
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		r, err := c.ListUsers(100)
		if err != nil {
			die(err)
		}
		outputJSON(r)
	},
}

var notionSearch = cobra.Command{
	Use:   "search [query]",
	Short: "Search pages and databases",
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		query := strings.Join(args, " ")
		r, err := c.Search(query, 100)
		if err != nil {
			die(err)
		}
		outputJSON(r)
	},
}

var notionPage = cobra.Command{
	Use:   "page",
	Short: "Page operations",
}

var notionPageGet = cobra.Command{
	Use:   "get <page-id>",
	Short: "Get page metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		p, err := c.GetPage(args[0])
		if err != nil {
			die(err)
		}
		outputJSON(p)
	},
}

var notionPageMarkdown = cobra.Command{
	Use:   "markdown <page-id>",
	Short: "Get page as Notion-flavored Markdown",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		md, err := c.GetPageMarkdown(args[0])
		if err != nil {
			die(err)
		}
		outputText(md)
	},
}

var notionPageCreate = cobra.Command{
	Use:   "create",
	Short: "Create a page under --parent or --db",
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		md := notionContent
		if md == "" && notionFile != "" {
			b, err := os.ReadFile(notionFile)
			if err != nil {
				die(err)
			}
			md = string(b)
		}
		if notionParent == "" && notionDBID == "" {
			die(fmt.Errorf("--parent or --db is required"))
		}
		p, err := c.CreatePage(parentRef(), marshalProps(), md)
		if err != nil {
			die(err)
		}
		outputJSON(p)
	},
}

var notionPageUpdate = cobra.Command{
	Use:   "update <page-id>",
	Short: "Update page properties",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		props := marshalProps()
		if notionContent != "" {
			if err := c.UpdatePageMarkdown(args[0], notionContent); err != nil {
				die(err)
			}
		}
		p, err := c.UpdatePageProperties(args[0], props)
		if err != nil {
			die(err)
		}
		outputJSON(p)
	},
}

var notionPageAppend = cobra.Command{
	Use:   "append <page-id>",
	Short: "Append blocks from --file or --content",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		content := notionContent
		if content == "" && notionFile != "" {
			b, err := os.ReadFile(notionFile)
			if err != nil {
				die(err)
			}
			content = string(b)
		}
		if content == "" {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				die(err)
			}
			content = string(b)
		}
		children := markdownToBlocks(content)
		_, err := c.AppendBlockChildren(args[0], children)
		if err != nil {
			die(err)
		}
		fmt.Println("Appended blocks")
	},
}

var notionDBID = cobra.Command{
	Use:   "db",
	Short: "Database operations",
}

var notionDbGet = cobra.Command{
	Use:   "get <database-id>",
	Short: "Get database metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		d, err := c.GetDatabase(args[0])
		if err != nil {
			die(err)
		}
		outputJSON(d)
	},
}

var notionDbQuery = cobra.Command{
	Use:   "query <database-id>",
	Short: "Query database entries",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		var filter, sorts json.RawMessage
		if notionQuery != "" {
			filter = json.RawMessage(notionQuery)
		}
		r, err := c.QueryDatabase(args[0], filter, sorts, 100)
		if err != nil {
			die(err)
		}
		outputJSON(r)
	},
}

var notionDbCreate = cobra.Command{
	Use:   "create",
	Short: "Create a database under --parent",
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		if notionParent == "" {
			die(fmt.Errorf("--parent is required"))
		}
		props := json.RawMessage(`{"Name":{"title":{}},"Status":{"select":{"options":[{"name":"Todo"},{"name":"Done"}]}}}`)
		if notionContent != "" {
			props = json.RawMessage(notionContent)
		}
		d, err := c.CreateDatabase(parentRef(), notionTitle, props, true)
		if err != nil {
			die(err)
		}
		outputJSON(d)
	},
}

var notionBlocks = cobra.Command{
	Use:   "blocks <block-or-page-id>",
	Short: "List block children",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		r, err := c.GetBlockChildren(args[0])
		if err != nil {
			die(err)
		}
		outputJSON(r)
	},
}

var notionFileUpload = cobra.Command{
	Use:   "upload <path>",
	Short: "Upload a file to Notion",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := notion.NewClient()
		path := args[0]
		b, err := os.ReadFile(path)
		if err != nil {
			die(err)
		}
		ext := strings.ToLower(filepath.Ext(path))
		ct := notion.contentTypeForExt(ext)
		upload, err := c.CreateFileUpload(filepath.Base(path), ct)
		if err != nil {
			die(err)
		}
		if err := c.UploadFileBytes(upload.URL, b, ct); err != nil {
			die(err)
		}
		outputJSON(upload)
	},
}

// notion.contentTypeForExt maps common extensions to MIME types.
func notion.contentTypeForExt(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// markdownToBlocks converts simple Markdown text into Notion blocks.
func markdownToBlocks(md string) []notion.Block {
	var blocks []notion.Block
	lines := strings.Split(md, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "# "):
			blocks = append(blocks, notion.Block{Object: "block", Type: "heading_1", Heading1: &notion.HeadingBlock{notion.RichText: []notion.RichText{{Type: "text", Text: &notion.TextContent{Content: strings.TrimPrefix(line, "# ")}}}}})
		case strings.HasPrefix(line, "## "):
			blocks = append(blocks, notion.Block{Object: "block", Type: "heading_2", Heading2: &notion.HeadingBlock{notion.RichText: []notion.RichText{{Type: "text", Text: &notion.TextContent{Content: strings.TrimPrefix(line, "## ")}}}}})
		case strings.HasPrefix(line, "- "):
			blocks = append(blocks, notion.Block{Object: "block", Type: "bulleted_list_item", BulletedListItem: &notion.ParagraphBlock{notion.RichText: []notion.RichText{{Type: "text", Text: &notion.TextContent{Content: strings.TrimPrefix(line, "- ")}}}}})
		case strings.HasPrefix(line, "[ ] "):
			blocks = append(blocks, notion.Block{Object: "block", Type: "to_do", ToDo: &notion.ToDoBlock{notion.RichText: []notion.RichText{{Type: "text", Text: &notion.TextContent{Content: strings.TrimPrefix(line, "[ ] ")}}}, Checked: false}})
		case strings.HasPrefix(line, "[x] "):
			blocks = append(blocks, notion.Block{Object: "block", Type: "to_do", ToDo: &notion.ToDoBlock{notion.RichText: []notion.RichText{{Type: "text", Text: &notion.TextContent{Content: strings.TrimPrefix(line, "[x] ")}}}, Checked: true}})
		default:
			blocks = append(blocks, notion.Block{Object: "block", Type: "paragraph", Paragraph: &notion.ParagraphBlock{notion.RichText: []notion.RichText{{Type: "text", Text: &notion.TextContent{Content: line}}}}})
		}
	}
	if len(blocks) == 0 {
		blocks = append(blocks, notion.Block{Object: "block", Type: "paragraph", Paragraph: &notion.ParagraphBlock{notion.RichText: []notion.RichText{{Type: "text", Text: &notion.TextContent{Content: ""}}}}})
	}
	return blocks
}

func init() {
	Notion.AddCommand(&notionMe)
	Notion.AddCommand(&notionUsers)
	Notion.AddCommand(&notionSearch)
	Notion.AddCommand(&notionPage)
	Notion.AddCommand(&notionDBID)
	Notion.AddCommand(&notionBlocks)
	Notion.AddCommand(&notionFileUpload)

	notionPage.AddCommand(&notionPageGet)
	notionPage.AddCommand(&notionPageMarkdown)
	notionPage.AddCommand(&notionPageCreate)
	notionPage.AddCommand(&notionPageUpdate)
	notionPage.AddCommand(&notionPageAppend)

	notionDBID.AddCommand(&notionDbGet)
	notionDBID.AddCommand(&notionDbQuery)
	notionDBID.AddCommand(&notionDbCreate)

	Notion.PersistentFlags().StringVarP(&notionOutput, "output", "o", "", "Write JSON output to file")
	Notion.PersistentFlags().StringVarP(&notionTitle, "title", "t", "", "Page/database title")
	Notion.PersistentFlags().StringVarP(&notionParent, "parent", "p", "", "Parent page ID")
	Notion.PersistentFlags().StringVarP(&notionDBID, "db", "d", "", "Database ID for create/query")
	Notion.PersistentFlags().StringVarP(&notionFile, "file", "f", "", "Read content from file")
	Notion.PersistentFlags().StringVarP(&notionContent, "content", "c", "", "Content/body as string")
	notionDbQuery.Flags().StringVarP(&notionQuery, "filter", "q", "", "JSON filter for query")
}
