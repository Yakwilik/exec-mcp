package mcp

import (
	"fmt"

	"github.com/Yakwilik/exec-mcp/internal/handler"
	"github.com/Yakwilik/exec-mcp/internal/tools"
	"github.com/Yakwilik/exec-mcp/internal/tools/exec"
	"github.com/Yakwilik/exec-mcp/internal/tools/run"
	"github.com/Yakwilik/exec-mcp/internal/tools/stop"
	execmcpv1 "github.com/Yakwilik/exec-mcp/proto"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateServer creates and configures the MCP server using protoc-gen-mcp generated code.
func CreateServer() (*mcp.Server, error) {
	server := mcp.NewServer(&mcp.Implementation{Name: "github.com/Yakwilik/exec-mcp", Version: "1.0.0"}, nil)

	if err := execmcpv1.RegisterExecMcpAPITools(server, handler.New()); err != nil {
		return nil, fmt.Errorf("register MCP tools: %w", err)
	}

	return server, nil
}

// GetTools returns all available tools.
// Kept for backward compatibility with tests.
func GetTools() []tools.Tool {
	return []tools.Tool{
		exec.NewExecProcessTool(),
		stop.NewStopProcessTool(),
		run.NewRunCommandTool(),
	}
}
