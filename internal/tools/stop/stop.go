package stop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Args represents the arguments for stop_process tool
type Args struct {
	PID  int  `json:"pid" jsonschema:"The process ID to stop"`
	Kill bool `json:"kill,omitempty" jsonschema:"If true, use SIGKILL instead of SIGTERM (force kill)"`
}

// StopProcessTool represents the stop_process tool
type StopProcessTool struct{}

// Name returns the tool name
func (t *StopProcessTool) Name() string {
	return "stop_process"
}

// Description returns the tool description
func (t *StopProcessTool) Description() string {
	return "Stop a process by its PID using SIGTERM or SIGKILL"
}

// Handle processes the tool call
func (t *StopProcessTool) Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
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

// NewStopProcessTool creates a new stop_process tool instance
func NewStopProcessTool() *StopProcessTool {
	return &StopProcessTool{}
}