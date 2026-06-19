# Architecture

## Overview

down.lsp follows a layered architecture with clear separation between:

1. **CLI Layer** (`cmd/`) — User-facing commands via cobra
2. **LSP Layer** (`lsp/`) — Language Server Protocol implementation
3. **Core Layer** (`core/`) — Domain models and business logic
4. **Internal Layer** (`internal/`) — Implementation details

## Key Design Patterns

### Handler-as-State

The LSP server uses a `State` struct that holds all runtime state and implements LSP method handlers as receiver methods:

```go
type State struct {
    Session     Session
    Server      *server.Server
    Workspaces  map[string]workspace.Workspace
    Diagnostics []protocol.Diagnostic
    Graph       *knowledge.Graph
    AI          *ai.Engine
    Documents   map[string]string
}
```

### Knowledge-First Architecture

On workspace initialization, all markdown files are scanned into a knowledge graph. This graph powers:
- Completions (entity suggestions)
- Hover (entity details)
- Diagnostics (broken link detection)
- Document symbols
- References/definitions
- AI context enrichment

### Pluggable AI Providers

The AI system uses an interface pattern for provider abstraction:

```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Name() string
    Available() bool
}
```

Providers are auto-detected via environment variables at startup.

### Generic Store

A typed generic store (`Store[K, V]`) is used throughout for flexible key-value storage with scope management (local/global/workspace/project/user/system).

### Concurrent Knowledge Graph

The knowledge graph uses `sync.RWMutex` for thread-safe concurrent access, supporting:
- Concurrent document scanning at startup
- Background re-indexing on file changes
- Thread-safe reads for completion/hover/etc.

## Data Flow

```
Document Change → DidChange/DidOpen/DidSave
    ↓
Knowledge Extraction → Graph Update
    ↓
Diagnostics Published → Client Notification
    ↓
Completions/Hover → Graph + AI Engine Query
```

## Module Dependencies

```
cmd/ → core/, lsp/
lsp/handler/ → lsp/knowledge/, lsp/ai/, core/workspace/
lsp/knowledge/ → (self-contained)
lsp/ai/ → lsp/knowledge/ (for context)
core/workspace/ → core/store/
core/profile/ → (self-contained)
```
