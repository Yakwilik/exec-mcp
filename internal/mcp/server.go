package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"exec-mcp/internal/tools"
	"exec-mcp/internal/tools/exec"
	"exec-mcp/internal/tools/stop"
)

// CreateServer creates and configures the MCP server
func CreateServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "exec-mcp", Version: "1.0.0"}, nil)

	// Create tool instances
	execTool := exec.NewExecProcessTool()
	stopTool := stop.NewStopProcessTool()

	// Add tools to server using the generic AddTool with proper schema
	mcp.AddTool(server, &mcp.Tool{
		Name:        execTool.Name(),
		Description: execTool.Description(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args exec.Args) (*mcp.CallToolResult, any, error) {
		return execTool.Handle(ctx, req)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        stopTool.Name(),
		Description: stopTool.Description(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args stop.Args) (*mcp.CallToolResult, any, error) {
		return stopTool.Handle(ctx, req)
	})

	return server
}

// GetTools returns all available tools
func GetTools() []tools.Tool {
	return []tools.Tool{
		exec.NewExecProcessTool(),
		stop.NewStopProcessTool(),
	}
}