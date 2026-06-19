package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	apiVersion = "2025-09-03"
	baseURL    = "https://api.notion.com/v1"
)

// Client is a thin Notion API v1 client using integration tokens.
type Client struct {
	APIKey string
	Base   string
	Client *http.Client
}

// NewClient returns a Notion client using NOTION_API_KEY or NOTION_API_TOKEN.
func NewClient() *Client {
	key := os.Getenv("NOTION_API_KEY")
	if key == "" {
		key = os.Getenv("NOTION_API_TOKEN")
	}
	return &Client{
		APIKey: key,
		Base:   baseURL,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

// HasKey reports whether an API key is configured.
func (c *Client) HasKey() bool { return c.APIKey != "" }

func (c *Client) request(method, path string, body any) (*http.Response, error) {
	if !c.HasKey() {
		return nil, fmt.Errorf("missing NOTION_API_KEY (or NOTION_API_TOKEN)")
	}
	url := c.Base + path
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Notion-Version", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.Client.Do(req)
}

func (c *Client) doJSON(method, path string, body any, out any) error {
	resp, err := c.request(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notion %s %s -> %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// SearchResult is the paginated wrapper returned by /search.
type SearchResult struct {
	Object  string `json:"object"`
	Results []struct {
		Object   string          `json:"object"`
		ID       string          `json:"id"`
		URL      string          `json:"url,omitempty"`
		Properties json.RawMessage `json:"properties,omitempty"`
		Title    []RichText      `json:"title,omitempty"`
	} `json:"results"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// TextContent is the inline text payload used inside RichText.
type TextContent struct {
	Content string `json:"content"`
	Link    *struct {
		URL string `json:"url"`
	} `json:"link,omitempty"`
}

// RichText is a Notion rich_text object.
type RichText struct {
	Type string `json:"type,omitempty"`
	Text *TextContent `json:"text,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	PlainText   string     `json:"plain_text,omitempty"`
}

// Annotations for rich text.
type Annotations struct {
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Code          bool   `json:"code,omitempty"`
	Color         string `json:"color,omitempty"`
}

// Search performs a paginated search.
func (c *Client) Search(query string, pageSize int) (*SearchResult, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	body := map[string]any{
		"query":     query,
		"page_size": pageSize,
	}
	out := &SearchResult{}
	return out, c.doJSON("POST", "/search", body, out)
}

// Page represents a Notion page object.
type Page struct {
	Object     string          `json:"object"`
	ID         string          `json:"id"`
	CreatedTime string         `json:"created_time"`
	LastEditedTime string      `json:"last_edited_time"`
	CreatedBy  struct{ ID string `json:"id"` } `json:"created_by"`
	LastEditedBy struct{ ID string `json:"id"` } `json:"last_edited_by"`
	Cover      json.RawMessage `json:"cover"`
	Icon       json.RawMessage `json:"icon"`
	Parent     json.RawMessage `json:"parent"`
	Archived   bool            `json:"archived"`
	InTrash    bool            `json:"in_trash"`
	Properties json.RawMessage `json:"properties"`
	URL        string          `json:"url"`
	PublicURL  string          `json:"public_url,omitempty"`
}

// GetPage fetches page metadata.
func (c *Client) GetPage(id string) (*Page, error) {
	id = normalizeID(id)
	out := &Page{}
	return out, c.doJSON("GET", "/pages/"+id, nil, out)
}

// GetPageMarkdown fetches a page rendered as Notion-flavored Markdown.
func (c *Client) GetPageMarkdown(id string) (string, error) {
	id = normalizeID(id)
	resp, err := c.request("GET", "/pages/"+id+"/markdown", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("notion markdown %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return string(data), nil
}

// PageCreate is the payload for creating a page.
type PageCreate struct {
	Parent     json.RawMessage `json:"parent"`
	Properties json.RawMessage `json:"properties"`
	Children   []Block         `json:"children,omitempty"`
	Markdown   string          `json:"markdown,omitempty"`
}

// CreatePage creates a page.
func (c *Client) CreatePage(parent map[string]any, props json.RawMessage, markdown string) (*Page, error) {
	body := map[string]any{
		"parent":     parent,
		"properties": props,
	}
	if markdown != "" {
		body["markdown"] = markdown
	}
	out := &Page{}
	return out, c.doJSON("POST", "/pages", body, out)
}

// UpdatePageProperties patches page properties.
func (c *Client) UpdatePageProperties(id string, props json.RawMessage) (*Page, error) {
	id = normalizeID(id)
	body := map[string]any{"properties": props}
	out := &Page{}
	return out, c.doJSON("PATCH", "/pages/"+id, body, out)
}

// UpdatePageMarkdown patches a page body via the markdown shortcut endpoint.
func (c *Client) UpdatePageMarkdown(id, markdown string) error {
	id = normalizeID(id)
	body := map[string]any{"markdown": markdown}
	return c.doJSON("PATCH", "/pages/"+id+"/markdown", body, nil)
}

// Database represents a data_source.
type Database struct {
	Object      string          `json:"object"`
	ID          string          `json:"id"`
	DataSourceID string         `json:"data_source_id,omitempty"`
	CreatedTime string          `json:"created_time"`
	LastEditedTime string       `json:"last_edited_time"`
	Title       []RichText      `json:"title"`
	Description []RichText      `json:"description"`
	Properties  json.RawMessage `json:"properties"`
	IsInline    bool            `json:"is_inline"`
	URL         string          `json:"url"`
	PublicURL   string          `json:"public_url,omitempty"`
}

// GetDatabase fetches a database/data source by ID.
func (c *Client) GetDatabase(id string) (*Database, error) {
	id = normalizeID(id)
	out := &Database{}
	return out, c.doJSON("GET", "/data_sources/"+id, nil, out)
}

// CreateDatabase creates a new data source.
func (c *Client) CreateDatabase(parent map[string]any, title string, props json.RawMessage, inline bool) (*Database, error) {
	body := map[string]any{
		"parent":     parent,
		"properties": props,
		"is_inline":  inline,
	}
	if title != "" {
		body["title"] = []map[string]any{
			{
				"type": "text",
				"text": map[string]any{"content": title},
			},
		}
	}
	out := &Database{}
	return out, c.doJSON("POST", "/data_sources", body, out)
}

// QueryDatabaseResult wraps a data_source query response.
type QueryDatabaseResult struct {
	Object     string          `json:"object"`
	Results    []json.RawMessage `json:"results"`
	NextCursor string          `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

// QueryDatabase queries a data source.
func (c *Client) QueryDatabase(id string, filter, sorts json.RawMessage, pageSize int) (*QueryDatabaseResult, error) {
	id = normalizeID(id)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	body := map[string]any{"page_size": pageSize}
	if len(filter) > 0 {
		body["filter"] = filter
	}
	if len(sorts) > 0 {
		body["sorts"] = sorts
	}
	out := &QueryDatabaseResult{}
	return out, c.doJSON("POST", "/data_sources/"+id+"/query", body, out)
}

// Block represents a Notion content block.
type Block struct {
	Object           string          `json:"object"`
	ID               string          `json:"id,omitempty"`
	Type             string          `json:"type"`
	Children         []Block         `json:"children,omitempty"`
	Paragraph        *ParagraphBlock `json:"paragraph,omitempty"`
	Heading1         *HeadingBlock   `json:"heading_1,omitempty"`
	Heading2         *HeadingBlock   `json:"heading_2,omitempty"`
	Heading3         *HeadingBlock   `json:"heading_3,omitempty"`
	BulletedListItem *ParagraphBlock `json:"bulleted_list_item,omitempty"`
	NumberedListItem *ParagraphBlock `json:"numbered_list_item,omitempty"`
	ToDo             *ToDoBlock      `json:"to_do,omitempty"`
	Code             *CodeBlock      `json:"code,omitempty"`
	Quote            *ParagraphBlock `json:"quote,omitempty"`
	Callout          *CalloutBlock   `json:"callout,omitempty"`
	Divider          *struct{}       `json:"divider,omitempty"`
}

// ParagraphBlock is shared by paragraph/list/quote.
type ParagraphBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
}

// HeadingBlock represents heading blocks.
type HeadingBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
}

// ToDoBlock represents a to_do block.
type ToDoBlock struct {
	RichText []RichText `json:"rich_text"`
	Checked  bool       `json:"checked"`
	Color    string     `json:"color,omitempty"`
}

// CodeBlock represents a code block.
type CodeBlock struct {
	RichText []RichText `json:"rich_text"`
	Language string     `json:"language,omitempty"`
}

// CalloutBlock represents a callout block.
type CalloutBlock struct {
	RichText []RichText `json:"rich_text"`
	Icon     json.RawMessage `json:"icon,omitempty"`
	Color    string     `json:"color,omitempty"`
}

// ChildrenResult holds block children.
type ChildrenResult struct {
	Object     string  `json:"object"`
	Results    []Block `json:"results"`
	NextCursor string  `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// GetBlockChildren fetches children of a block/page.
func (c *Client) GetBlockChildren(id string) (*ChildrenResult, error) {
	id = normalizeID(id)
	out := &ChildrenResult{}
	return out, c.doJSON("GET", "/blocks/"+id+"/children", nil, out)
}

// AppendBlockChildren appends blocks to a page or block.
func (c *Client) AppendBlockChildren(id string, children []Block) (*ChildrenResult, error) {
	id = normalizeID(id)
	body := map[string]any{"children": children}
	out := &ChildrenResult{}
	return out, c.doJSON("PATCH", "/blocks/"+id+"/children", body, out)
}

// User represents a Notion user.
type User struct {
	Object    string `json:"object"`
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Type      string `json:"type,omitempty"`
	Person    *struct {
		Email string `json:"email"`
	} `json:"person,omitempty"`
	Bot *struct{} `json:"bot,omitempty"`
}

// ListUsersResult is the paginated users response.
type ListUsersResult struct {
	Results    []User `json:"results"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// ListUsers returns workspace users.
func (c *Client) ListUsers(pageSize int) (*ListUsersResult, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	out := &ListUsersResult{}
	return out, c.doJSON("GET", fmt.Sprintf("/users?page_size=%d", pageSize), nil, out)
}

// GetMe returns the bot user.
func (c *Client) GetMe() (*User, error) {
	out := &User{}
	return out, c.doJSON("GET", "/users/me", nil, out)
}

// FileUploadResponse is the first step of file upload flow.
type FileUploadResponse struct {
	URL            string `json:"upload_url"`
	SignedURLExpires int `json:"signed_url_expires"`
	FileUploadID   string `json:"file_upload_id"`
}

// CreateFileUpload creates an upload request for a file.
func (c *Client) CreateFileUpload(name, contentType string) (*FileUploadResponse, error) {
	body := map[string]any{
		"filename":     name,
		"content_type": contentType,
	}
	out := &FileUploadResponse{}
	return out, c.doJSON("POST", "/file_uploads", body, out)
}

// UploadFileBytes PUTs raw bytes to the upload URL.
func (c *Client) UploadFileBytes(uploadURL string, data []byte, contentType string) error {
	req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// normalizeID strips dashes from UUIDs and lowercases.
func normalizeID(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "-", ""))
}
