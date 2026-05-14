package main

import (
	"context"
	"log"

	servermcp "github.com/Yakwilik/exec-mcp/internal/mcp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Create the MCP server
	server, err := servermcp.CreateServer()
	if err != nil {
		log.Fatalf("failed to create MCP server: %v", err)
	}

	// Run the server on stdio transport
	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
