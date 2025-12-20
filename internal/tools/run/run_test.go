package run

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandTool(t *testing.T) {
	tool := NewRunCommandTool()

	t.Run("tool name and description", func(t *testing.T) {
		assert.Equal(t, "run_command", tool.Name())
		assert.Equal(t, "Execute a command and return its output immediately (synchronous execution)", tool.Description())
	})

	t.Run("successful command execution", func(t *testing.T) {
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
			args = []string{"/c", "echo", "test output"}
		} else {
			command = "echo"
			args = []string{"test output"}
		}

		// Set up request arguments
		argsMap := map[string]interface{}{
			"command": command,
			"args":    args,
		}
		argsJSON, err := json.Marshal(argsMap)
		require.NoError(t, err)
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify result content
		assert.Len(t, result.Content, 1)
		assert.IsType(t, &mcp.TextContent{}, result.Content[0])

		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Command executed")

		// Verify data structure
		runResult, ok := data.(Result)
		require.True(t, ok, "Expected Result type")
		assert.Equal(t, command, runResult.Command)
		assert.Equal(t, args, runResult.Args)
		assert.Equal(t, 0, runResult.ExitCode)
		assert.Contains(t, runResult.Stdout, "test output")
		assert.True(t, runResult.Success)
		assert.Greater(t, runResult.Duration, 0.0)
		assert.WithinDuration(t, time.Now(), runResult.ExecutedAt, 1*time.Second)
	})

	t.Run("command with non-zero exit code", func(t *testing.T) {
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
			args = []string{"/c", "exit", "1"}
		} else {
			command = "sh"
			args = []string{"-c", "exit 1"}
		}

		// Set up request arguments
		argsMap := map[string]interface{}{
			"command": command,
			"args":    args,
		}
		argsJSON, err := json.Marshal(argsMap)
		require.NoError(t, err)
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		// Should not return an error (tool executed successfully, command failed)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify result indicates failure
		assert.True(t, result.IsError)

		// Verify data structure
		runResult, ok := data.(Result)
		require.True(t, ok)
		assert.Equal(t, 1, runResult.ExitCode)
		assert.False(t, runResult.Success)
	})

	t.Run("command with timeout", func(t *testing.T) {
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
			args = []string{"/c", "timeout", "5"}
		} else {
			command = "sleep"
			args = []string{"5"}
		}

		// Set up request arguments with 1 second timeout
		argsMap := map[string]interface{}{
			"command": command,
			"args":    args,
			"timeout": 1,
		}
		argsJSON, err := json.Marshal(argsMap)
		require.NoError(t, err)
		req.Params.Arguments = json.RawMessage(argsJSON)

		start := time.Now()
		result, data, err := tool.Handle(ctx, req)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify timeout was respected (should complete in ~1 second, not 5)
		assert.Less(t, elapsed, 2*time.Second, "Command should be killed after timeout")

		// Verify result indicates failure due to timeout
		runResult, ok := data.(Result)
		require.True(t, ok)
		assert.Equal(t, -1, runResult.ExitCode) // -1 indicates context/timeout error
		assert.False(t, runResult.Success)
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

		// Should not return an error (tool executed successfully, command failed)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify result indicates failure
		assert.True(t, result.IsError)

		// Verify data structure
		runResult, ok := data.(Result)
		require.True(t, ok)
		assert.Equal(t, -1, runResult.ExitCode) // -1 indicates command not found or other error
		assert.False(t, runResult.Success)
		assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "Command failed")
	})
}

func TestResultJSONSerialization(t *testing.T) {
	result := Result{
		Command:    "test-command",
		Args:       []string{"arg1", "arg2"},
		ExitCode:   0,
		Stdout:     "test output",
		Stderr:     "",
		ExecutedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Duration:   1.5,
		Success:    true,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(result)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled Result
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, result, unmarshaled)
}
