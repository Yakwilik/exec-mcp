package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Args represents the arguments for exec_process tool
type Args struct {
	Command string   `json:"command" jsonschema:"The command to execute"`
	Args    []string `json:"args,omitempty" jsonschema:"Command line arguments"`
	Dir     string   `json:"dir,omitempty" jsonschema:"Working directory for the process"`
	Env     []string `json:"env,omitempty" jsonschema:"Environment variables (format: KEY=VALUE)"`
}

// ProcessInfo holds information about a running process
type ProcessInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
}

// ExecProcessTool executes a process and returns its PID
func ExecProcessTool(ctx context.Context, req *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, any, error) {
	if args.Command == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Command is required"},
			},
		}, nil, fmt.Errorf("command is required")
	}

	// Create command
	cmd := exec.CommandContext(ctx, args.Command, args.Args...)
	
	// Set working directory if provided
	if args.Dir != "" {
		cmd.Dir = args.Dir
	}
	
	// Set environment variables if provided
	if len(args.Env) > 0 {
		cmd.Env = append(os.Environ(), args.Env...)
	}

	// Start the process (detached)
	if err := cmd.Start(); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error starting process: %v", err)},
			},
		}, nil, err
	}

	// Create process info
	processInfo := ProcessInfo{
		PID:       cmd.Process.Pid,
		Command:   args.Command,
		Args:      args.Args,
		StartTime: time.Now(),
		Status:    "running",
	}

	// Convert to JSON for response
	resultJSON, err := json.MarshalIndent(processInfo, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error serializing result: %v", err)},
			},
		}, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Process started successfully:\n%s", string(resultJSON))},
		},
	}, processInfo, nil
}