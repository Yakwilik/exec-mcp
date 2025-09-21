package main

import (
	"context"
	"log"

	servermcp "exec-mcp/internal/mcp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Create the MCP server
	server := servermcp.CreateServer()

	// Run the server on stdio transport
	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}