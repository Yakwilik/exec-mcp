package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProcessManager handles process execution and management
type ProcessManager struct {
	processes map[int]*exec.Cmd
}

// ProcessInfo holds information about a running process
type ProcessInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
}

// NewProcessManager creates a new process manager
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[int]*exec.Cmd),
	}
}

// ExecProcessArgs represents the arguments for exec_process tool
type ExecProcessArgs struct {
	Command string   `json:"command" jsonschema:"The command to execute"`
	Args    []string `json:"args,omitempty" jsonschema:"Command line arguments"`
	Dir     string   `json:"dir,omitempty" jsonschema:"Working directory for the process"`
	Env     []string `json:"env,omitempty" jsonschema:"Environment variables (format: KEY=VALUE)"`
}

// StopProcessArgs represents the arguments for stop_process tool
type StopProcessArgs struct {
	PID  int  `json:"pid" jsonschema:"The process ID to stop"`
	Kill bool `json:"kill,omitempty" jsonschema:"If true, use SIGKILL instead of SIGTERM (force kill)"`
}

// ExecProcessTool executes a process and returns its PID
func ExecProcessTool(ctx context.Context, req *mcp.CallToolRequest, args ExecProcessArgs) (*mcp.CallToolResult, any, error) {
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

	// Start the process
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

// StopProcessTool stops a process by PID
func StopProcessTool(ctx context.Context, req *mcp.CallToolRequest, args StopProcessArgs) (*mcp.CallToolResult, any, error) {
	if args.PID == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "PID is required"},
			},
		}, nil, fmt.Errorf("PID is required")
	}

	// Find the process by PID
	process, err := os.FindProcess(args.PID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Process with PID %d not found: %v", args.PID, err)},
			},
		}, nil, err
	}

	// Determine signal to send
	var signal syscall.Signal
	if args.Kill {
		signal = syscall.SIGKILL
	} else {
		signal = syscall.SIGTERM
	}

	// Send signal to process
	if err := process.Signal(signal); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error sending signal to process %d: %v", args.PID, err)},
			},
		}, nil, err
	}

	signalType := "SIGTERM"
	if args.Kill {
		signalType = "SIGKILL"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Signal %s sent to process %d", signalType, args.PID)},
		},
	}, map[string]interface{}{
		"pid":    args.PID,
		"signal": signalType,
		"status": "signal_sent",
	}, nil
}

// CreateServer creates and configures the MCP server
func CreateServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "exec-mcp", Version: "1.0.0"}, nil)

	// Add exec_process tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "exec_process",
		Description: "Execute a process and return its PID for later management",
	}, ExecProcessTool)

	// Add stop_process tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_process",
		Description: "Stop a process by its PID using SIGTERM or SIGKILL",
	}, StopProcessTool)

	return server
}