package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecProcessTool(t *testing.T) {
	tool := NewExecProcessTool()

	t.Run("tool name and description", func(t *testing.T) {
		assert.Equal(t, "exec_process", tool.Name())
		assert.Equal(t, "Execute a process and return its PID for later management", tool.Description())
	})

	t.Run("successful process execution", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "echo", "test"}
		} else {
			command = "echo"
			args = []string{"test"}
		}

		// Set up request arguments
		argsJSON := fmt.Sprintf(`{"command": "%s", "args": ["%s"]}`, command, args[0])
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify result content
		assert.Len(t, result.Content, 1)
		assert.IsType(t, &mcp.TextContent{}, result.Content[0])
		
		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Process started successfully")

		// Verify data structure
		processInfo, ok := data.(ProcessInfo)
		require.True(t, ok, "Expected ProcessInfo type")
		assert.Greater(t, processInfo.PID, 0)
		assert.Equal(t, command, processInfo.Command)
		assert.Equal(t, args, processInfo.Args)
		assert.Equal(t, "running", processInfo.Status)
		assert.WithinDuration(t, time.Now(), processInfo.StartTime, 1*time.Second)
	})

	t.Run("long running process (detached)", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "ping", "127.0.0.1", "-n", "10"}
		} else {
			command = "sleep"
			args = []string{"10"}
		}

		// Set up request arguments
		var argsJSON string
		if runtime.GOOS == "windows" {
			argsJSON = fmt.Sprintf(`{"command": "%s", "args": ["%s", "%s", "%s", "%s"]}`, command, args[0], args[1], args[2], args[3])
		} else {
			argsJSON = fmt.Sprintf(`{"command": "%s", "args": ["%s"]}`, command, args[0])
		}
		req.Params.Arguments = json.RawMessage(argsJSON)

		start := time.Now()
		result, data, err := tool.Handle(ctx, req)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify the function returns quickly (non-blocking)
		assert.Less(t, elapsed, 1*time.Second, "Function should return immediately, not wait for process completion")

		processInfo, ok := data.(ProcessInfo)
		require.True(t, ok)
		assert.Greater(t, processInfo.PID, 0)

		// Verify process is actually running and clean up
		process, err := os.FindProcess(processInfo.PID)
		require.NoError(t, err)
		process.Signal(syscall.SIGTERM)
	})

	t.Run("missing command", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		// Set up request with empty command
		argsJSON := `{"command": ""}`
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		// Should return an error
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, data)

		// Verify error content
		assert.Len(t, result.Content, 1)
		assert.IsType(t, &mcp.TextContent{}, result.Content[0])
		
		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Command is required")
	})

	t.Run("invalid command", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		// Set up request with invalid command
		argsJSON := `{"command": "nonexistent_command_12345"}`
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		// Should return an error
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, data)

		// Verify error content
		assert.Len(t, result.Content, 1)
		assert.IsType(t, &mcp.TextContent{}, result.Content[0])
		
		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Error starting process")
	})
}

func TestProcessInfoJSONSerialization(t *testing.T) {
	processInfo := ProcessInfo{
		PID:       12345,
		Command:   "test-command",
		Args:      []string{"arg1", "arg2"},
		StartTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Status:    "running",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(processInfo)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled ProcessInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, processInfo, unmarshaled)
}