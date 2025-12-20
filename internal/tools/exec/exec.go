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

// ExecProcessTool represents the exec_process tool
type ExecProcessTool struct{}

// Name returns the tool name
func (t *ExecProcessTool) Name() string {
	return "exec_process"
}

// Description returns the tool description
func (t *ExecProcessTool) Description() string {
	return "Execute a process and return its PID for later management"
}

// Handle processes the tool call
func (t *ExecProcessTool) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
	if req.Params == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Missing parameters"},
			},
		}, nil, fmt.Errorf("missing parameters")
	}

	// Parse arguments from the request
	var args Args
	if req.Params.Arguments != nil {
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error parsing arguments: %v", err)},
				},
			}, nil, err
		}
	}

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

// NewExecProcessTool creates a new exec_process tool instance
func NewExecProcessTool() *ExecProcessTool {
	return &ExecProcessTool{}
}