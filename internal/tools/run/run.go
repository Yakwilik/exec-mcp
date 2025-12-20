package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Args represents the arguments for run_command tool
type Args struct {
	Command string   `json:"command" jsonschema:"The command to execute"`
	Args    []string `json:"args,omitempty" jsonschema:"Command line arguments"`
	Dir     string   `json:"dir,omitempty" jsonschema:"Working directory for the process"`
	Env     []string `json:"env,omitempty" jsonschema:"Environment variables (format: KEY=VALUE)"`
	Timeout int      `json:"timeout,omitempty" jsonschema:"Timeout in seconds (0 means no timeout)"`
}

// Result holds the execution result
type Result struct {
	Command    string    `json:"command"`
	Args       []string  `json:"args"`
	ExitCode   int       `json:"exit_code"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	ExecutedAt time.Time `json:"executed_at"`
	Duration   float64   `json:"duration_seconds"`
	Success    bool      `json:"success"`
}

// RunCommandTool represents the run_command tool
type RunCommandTool struct{}

// Name returns the tool name
func (t *RunCommandTool) Name() string {
	return "run_command"
}

// Description returns the tool description
func (t *RunCommandTool) Description() string {
	return "Execute a command and return its output immediately (synchronous execution)"
}

// Handle processes the tool call
func (t *RunCommandTool) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
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

	// Create context with timeout if specified
	execCtx := ctx
	if args.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(args.Timeout)*time.Second)
		defer cancel()
	}

	// Create command
	cmd := exec.CommandContext(execCtx, args.Command, args.Args...)

	// Set working directory if provided
	if args.Dir != "" {
		cmd.Dir = args.Dir
	}

	// Set environment variables if provided
	if len(args.Env) > 0 {
		cmd.Env = append(os.Environ(), args.Env...)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Record start time
	startTime := time.Now()

	// Execute the command
	err := cmd.Run()
	executedAt := time.Now()
	duration := executedAt.Sub(startTime).Seconds()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Context timeout or other error
			exitCode = -1
		}
	}

	// Create result
	result := Result{
		Command:    args.Command,
		Args:       args.Args,
		ExitCode:   exitCode,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExecutedAt: executedAt,
		Duration:   duration,
		Success:    exitCode == 0,
	}

	// Convert to JSON for response
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error serializing result: %v", err)},
			},
			IsError: true,
		}, nil, err
	}

	// Build response text
	responseText := fmt.Sprintf("Command executed:\n%s", string(resultJSON))
	if exitCode != 0 {
		responseText = fmt.Sprintf("Command failed (exit code %d):\n%s", exitCode, string(resultJSON))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
		IsError: exitCode != 0,
	}, result, nil
}

// NewRunCommandTool creates a new run_command tool instance
func NewRunCommandTool() *RunCommandTool {
	return &RunCommandTool{}
}
