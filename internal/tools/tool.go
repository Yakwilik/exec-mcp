package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool defines the interface that all tools must implement
type Tool interface {
	// Name returns the tool name
	Name() string
	
	// Description returns the tool description
	Description() string
	
	// Handle processes the tool call
	Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error)
}