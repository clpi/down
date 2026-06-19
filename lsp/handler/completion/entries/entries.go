package entries

var (
	Emojis = map[string]string{
		"happy":      "😀",
		"sad":        "😢",
		"angry":      "😠",
		"confused":   "😕",
		"excited":    "😆",
		"love":       "😍",
		"laughing":   "😂",
		"crying":     "😭",
		"sleepy":     "😴",
		"surprised":  "😮",
		"sick":       "🤒",
		"cool":       "😎",
		"nerd":       "🤓",
		"worried":    "😟",
		"scared":     "😨",
		"silly":      "🤪",
		"shocked":    "😱",
		"sunglasses": "😎",
		"tongue":     "😛",
		"thinking":   "🤔",
	}
	CodeTemplates map[string]Template = map[string]Template{}
	CodeSnippets  map[string]string   = map[string]string{}
	Templates     map[string]Template = map[string]Template{
		"note": {
			Body:        "Note",
			Description: "Insert a note",
			Document:    "file:///path/to/document",
			Workspace:   "workspace",
			URI:         "file:///path/to/document",
		},
		"daily": {
			Body:        "Daily",
			Description: "Insert a daily note",
			Document:    "file:///path/to/document",
			Workspace:   "workspace",
			URI:         "file:///path/to/document",
		},
		"log": {
			Body:        "Log",
			Description: "Insert a log",
			Document:    "file:///path/to/document",
			Workspace:   "workspace",
			URI:         "file:///path/to/document",
		},
		"index": {
			Body:        "Index",
			Description: "Insert an index",
		},
	}
	Snippets map[string]Snippet = map[string]Snippet{
		"#date": {
			Body:        "date +%Y-%m-%d",
			Description: "Insert the current date in the format YYYY-MM-DD",
		},
		"#time": {
			Body:        "date +%H:%M:%S",
			Description: "Insert the current time in the format HH:MM:SS",
		},
		"#datetime": {
			Body:        "date",
			Description: "Insert the current date and time",
		},
	}
)
