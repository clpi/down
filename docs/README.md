# down.lsp Documentation

**down.lsp** is a Language Server Protocol (LSP) implementation for markdown-based note-taking and knowledge management. It provides intelligent completions, semantic analysis, knowledge graph construction, AI-powered writing assistance, and workspace management.

## Table of Contents

- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [LSP Features](#lsp-features)
- [Knowledge Base](#knowledge-base)
- [AI Integration](#ai-integration)
- [Workspace Management](#workspace-management)
- [User Profile](#user-profile)
- [CLI Commands](#cli-commands)
- [Configuration](#configuration)
- [Embedding & Fine-Tuning](#embedding--fine-tuning)

## Architecture

```
down.lsp/
├── cmd/             # CLI commands (cobra)
├── core/            # Core domain types
│   ├── config/      # Global configuration
│   ├── data/        # Data structures
│   ├── entities/    # Domain entities (Task, Tag, Link, etc.)
│   ├── fs/          # Filesystem utilities
│   ├── profile/     # User profile management
│   ├── store/       # Generic key-value store
│   └── workspace/   # Workspace management
├── internal/        # Internal utilities
├── lsp/             # LSP server implementation
│   ├── ai/          # AI providers and embeddings
│   ├── handler/     # LSP protocol handlers
│   │   ├── completion/  # Completion providers
│   │   └── semantic/    # Semantic tokens
│   ├── knowledge/   # Knowledge graph and base
│   └── files/       # File type definitions
└── docs/            # Documentation
```

## Getting Started

### Installation

```bash
go install github.com/clpi/down.lsp@latest
```

### Running the LSP Server

```bash
down lsp
```

The server communicates over stdio using JSON-RPC 2.0.

### Initialize a Workspace

```bash
down workspace init /path/to/notes
```

### Set Up User Profile

```bash
down profile init "Your Name"
```

## LSP Features

### Completions

- **Snippet completions**: Date/time, template insertions
- **Emoji completions**: Type emoji names for quick insertion
- **File completions**: Reference workspace files
- **HTML tag completions**: Inline HTML support
- **Wiki link completions**: `[[` triggers wiki link suggestions
- **Knowledge graph completions**: Entity names from the graph
- **AI completions**: Context-aware text suggestions

### Hover Information

Hovering over any word shows:
- Knowledge graph entity details (type, mentions, properties)
- Relations to other entities
- Back-references from other documents

### Diagnostics

- **Broken links**: File references that don't resolve
- **Unresolved wiki links**: `[[targets]]` without matching files
- **Overdue tasks**: Open tasks with past dates

### Code Actions

- Create link on cursor word
- Generate Table of Contents
- AI: Expand/Summarize/Explain selection
- Search knowledge graph
- Suggest next steps from open tasks

### Semantic Tokens

Full semantic highlighting for:
- Headings, tags (#), mentions (@)
- Wiki links, markdown links
- Tasks, code spans, dates
- Frontmatter keys, blockquotes
- Bold, italic formatting

### Document Symbols

All knowledge graph entities in a document appear as symbols, navigable via the outline view.

### Workspace Symbols

Search across all indexed entities in the workspace.

### Folding Ranges

- Heading sections fold at their level
- Code blocks fold between fences
- Frontmatter folds between `---` markers

### Selection Ranges

Smart selection expansion: word → line → paragraph → section → document.

### Inlay Hints

- Word count per section heading
- Task completion progress
- Wiki link mention counts

### Document Links

- Wiki links `[[target]]` resolve to workspace files
- Markdown links `[text](path)` resolve relative paths

### Linked Editing

Editing a wiki link `[[target]]` simultaneously updates all matching links in the document.

### Rename

Rename a word across all open documents with proper word-boundary matching.

### Formatting

- Trim trailing whitespace
- Normalize heading spacing
- Collapse multiple blank lines
- Normalize list item spacing
- Ensure final newline

## Knowledge Base

### Overview

The knowledge base automatically extracts entities and relations from your markdown documents:

**Entity Types:**
- `person` — @mentions and frontmatter authors
- `concept` — Headings and referenced topics
- `project` — Frontmatter project fields
- `action` — Task items
- `tag` — #hashtags and frontmatter tags
- `document` — Wiki links and file references
- `date` — Dates in YYYY-MM-DD format
- `code` — Inline code references
- `place` — Location references

**Relation Types:**
- `mentions`, `relates_to`, `depends_on`
- `created_by`, `tagged_with`, `links_to`
- `part_of`, `assigned_to`, `scheduled`, `blocks`

### Collections

Group related documents into collections for targeted search:

```
down.knowledge.collections  # List collections
```

### Semantic Search

The knowledge base supports keyword-based search with TF-IDF scoring across document chunks.

### Commands

| Command | Description |
|---------|-------------|
| `down.knowledge.summary` | Overview of the knowledge graph |
| `down.knowledge.search <query>` | Search entities |
| `down.knowledge.entities [kind]` | List entities by type |
| `down.knowledge.relations <entity>` | Show entity relations |
| `down.knowledge.related <uri>` | Find related documents |
| `down.knowledge.reindex` | Rebuild the knowledge graph |

## AI Integration

### Supported Providers

| Provider | Env Variables | Description |
|----------|--------------|-------------|
| Anthropic | `ANTHROPIC_API_KEY` | Claude models |
| Ollama | `DOWN_OLLAMA_BASE_URL` | Local models |
| Gemini | `GEMINI_API_KEY` | Google AI |
| xAI | `XAI_API_KEY` | Grok models |
| Cloudflare | `CLOUDFLARE_API_TOKEN` | Workers AI |

### Configuration

Set `DOWN_AI_PROVIDER` to explicitly select a provider, or let it auto-detect.

### Features

- **Text completion**: Context-aware line completions
- **Query**: Conversational Q&A with workspace knowledge
- **Suggest**: Related topic suggestions
- **Transform**: Expand, summarize, or explain text
- **Fine-tuning**: Train local embeddings on your corpus

### Commands

| Command | Description |
|---------|-------------|
| `down.ai.query <question>` | Ask AI a question |
| `down.ai.suggest [uri]` | Get related suggestions |
| `down.ai.expand <text>` | Expand selected text |
| `down.ai.summarize <text>` | Summarize text |
| `down.ai.explain <text>` | Explain text |
| `down.ai.providers` | Show provider status |
| `down.ai.clear` | Clear conversation history |
| `down.ai.finetune` | Generate fine-tuning data |

## Workspace Management

### Multi-Workspace Support

down.lsp supports multiple simultaneous workspaces with independent settings.

### Workspace Structure

```
workspace/
├── .down/           # Internal store
│   ├── settings.json
│   └── knowledge.json
├── notes/           # Notes directory
├── templates/       # Templates
├── snippets/        # Snippets
├── journal/         # Daily journal
├── attachments/     # File attachments
└── index.md         # Workspace index
```

### Workspace Settings

Settings are stored in `.down/settings.json`:

```json
{
  "index_name": "index.md",
  "notes_dir": "notes",
  "templates_dir": "templates",
  "journal_dir": "journal",
  "exclude_globs": [".git", "node_modules"],
  "include_globs": ["*.md", "*.markdown"],
  "ai_enabled": true,
  "auto_index": true
}
```

### Workspace Events

The system tracks workspace lifecycle events:
- `created`, `opened`, `closed`
- `file_changed`, `file_created`, `file_deleted`
- `reindexed`, `configured`

## User Profile

### Overview

User profiles store personal preferences, AI settings, and workspace references.

Profile is stored at `~/.down/profile.json`.

### CLI

```bash
down profile                    # Show profile
down profile init "Name"        # Create profile
down profile set theme dark     # Set preference
down profile set auto_save true # Toggle features
```

### Preferences

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `theme` | string | "default" | UI theme |
| `date_format` | string | "2006-01-02" | Date display format |
| `language` | string | "en" | Language code |
| `auto_save` | bool | true | Auto-save documents |
| `auto_index` | bool | true | Auto-index on save |
| `show_inlay_hints` | bool | true | Show inlay hints |
| `show_code_lens` | bool | true | Show code lenses |
| `max_completions` | int | 20 | Max completion items |
| `tab_size` | int | 2 | Tab width |
| `word_wrap` | bool | true | Word wrap |
| `spell_check` | bool | false | Spell checking |

## CLI Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `down lsp` | `ls`, `L` | Start LSP server |
| `down init` | - | Initialize workspace |
| `down workspace` | `ws` | Workspace management |
| `down profile` | `prof`, `user` | User profile |
| `down note` | - | Note operations |
| `down find` | - | Search documents |
| `down list` | - | List items |
| `down new` | - | Create items |
| `down log` | - | Log management |
| `down config` | - | Configuration |
| `down export` | - | Export data |
| `down sync` | - | Sync operations |
| `down serve` | - | HTTP server |
| `down shell` | - | Interactive shell |
| `down delete` | - | Delete items |

## Configuration

### Global Settings

Global config in `~/.down/config.json`.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DOWN_AI_PROVIDER` | Force AI provider |
| `DOWN_ANTHROPIC_MODEL` | Anthropic model name |
| `DOWN_OLLAMA_BASE_URL` | Ollama server URL |
| `DOWN_OLLAMA_MODEL` | Ollama model name |
| `DOWN_GEMINI_MODEL` | Gemini model name |
| `DOWN_XAI_MODEL` | xAI model name |
| `DOWN_CF_MODEL` | Cloudflare model name |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `GEMINI_API_KEY` | Google AI key |
| `XAI_API_KEY` | xAI API key |
| `CLOUDFLARE_API_TOKEN` | Cloudflare token |
| `CLOUDFLARE_ACCOUNT_ID` | Cloudflare account |

## Embedding & Fine-Tuning

### Local Embeddings

down.lsp includes a local bag-of-words embedding model that works offline:

- Hash-based dimensionality reduction (default: 384 dimensions)
- TF-IDF weighting from corpus statistics
- Cosine similarity for semantic search

### Fine-Tuning

Generate training pairs from your workspace and fine-tune the local model:

1. Open documents in the workspace
2. Run `down.ai.finetune` command
3. Training pairs are generated from heading-content pairs and cross-document similarity
4. Contrastive learning updates IDF weights

The fine-tuned model improves search relevance for your specific domain vocabulary.

### Training Data Generation

Automatic training pair sources:
- **Positive pairs**: Heading → content under that heading
- **Negative pairs**: Content from unrelated documents
- **Entity pairs**: Documents sharing knowledge graph entities
