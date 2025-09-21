package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"exec-mcp/internal/tools/exec"
	"exec-mcp/internal/tools/stop"
)

// CreateServer creates and configures the MCP server
func CreateServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "exec-mcp", Version: "1.0.0"}, nil)

	// Add exec_process tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "exec_process",
		Description: "Execute a process and return its PID for later management",
	}, exec.ExecProcessTool)

	// Add stop_process tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_process",
		Description: "Stop a process by its PID using SIGTERM or SIGKILL",
	}, stop.StopProcessTool)

	return server
}