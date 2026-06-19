package mcp

import (
	"context"
	"fmt"
	"log"

	cmdutil "github.com/clpi/down/cmd/util"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var mcpR = func(cmd *cobra.Command, args []string) {
	// We use stderr for logs because MCP communicates over stdout
	log.SetOutput(cmd.ErrOrStderr())
	log.Println("Starting down.nvim MCP server...")

	// Create MCP server
	s := server.NewMCPServer(
		"down.nvim",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Add tool: list_workspaces
	listWorkspacesTool := mcp.NewTool("list_workspaces",
		mcp.WithDescription("List all available down.nvim workspaces"),
	)
	s.AddTool(listWorkspacesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res := "Available workspaces tool not fully implemented yet."
		return mcp.NewToolResultText(res), nil
	})

	// Add tool: search_notes
	searchNotesTool := mcp.NewTool("search_notes",
		mcp.WithDescription("Search for notes in the current workspace"),
		mcp.WithString("query", mcp.Required(), mcp.Description("The search query")),
	)
	s.AddTool(searchNotesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := request.Arguments["query"].(string)
		res := fmt.Sprintf("Searching for %s in workspace... (not fully implemented)", query)
		return mcp.NewToolResultText(res), nil
	})

	// Add tool: read_note
	readNoteTool := mcp.NewTool("read_note",
		mcp.WithDescription("Read a specific note by path"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the markdown note")),
	)
	s.AddTool(readNoteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := request.Arguments["path"].(string)
		res := fmt.Sprintf("Reading note at %s... (not fully implemented)", path)
		return mcp.NewToolResultText(res), nil
	})

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

var Mcp = cmdutil.Cmd("mcp", []string{"m"}, "Model Context Protocol Server", "Run the MCP server", mcpR)
