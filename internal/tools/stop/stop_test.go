package stop

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopProcessTool(t *testing.T) {
	tool := NewStopProcessTool()

	t.Run("tool name and description", func(t *testing.T) {
		assert.Equal(t, "stop_process", tool.Name())
		assert.Equal(t, "Stop a process by its PID using SIGTERM or SIGKILL", tool.Description())
	})

	t.Run("successful process termination with SIGTERM", func(t *testing.T) {
		// Start a long-running process
		var cmd *exec.Cmd
		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "ping", "127.0.0.1", "-n", "10"}
		} else {
			command = "sleep"
			args = []string{"10"}
		}
		
		cmd = exec.Command(command, args...)
		err := cmd.Start()
		require.NoError(t, err)
		
		pid := cmd.Process.Pid
		
		// Wait a bit to ensure process is running
		time.Sleep(100 * time.Millisecond)

		// Now test the stop tool
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		// Set up request arguments
		argsJSON := fmt.Sprintf(`{"pid": %d, "kill": false}`, pid)
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify result content
		assert.Len(t, result.Content, 1)
		assert.IsType(t, &mcp.TextContent{}, result.Content[0])
		
		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Signal SIGTERM sent to process")
		assert.Contains(t, textContent.Text, fmt.Sprintf("%d", pid))

		// Verify data structure
		resultData, ok := data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, pid, resultData["pid"])
		assert.Equal(t, "SIGTERM", resultData["signal"])
		assert.Equal(t, "signal_sent", resultData["status"])

		// Wait for process to actually terminate
		cmd.Wait()
	})

	t.Run("successful process termination with SIGKILL", func(t *testing.T) {
		// Start a long-running process
		var cmd *exec.Cmd
		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "ping", "127.0.0.1", "-n", "10"}
		} else {
			command = "sleep"
			args = []string{"10"}
		}
		
		cmd = exec.Command(command, args...)
		err := cmd.Start()
		require.NoError(t, err)
		
		pid := cmd.Process.Pid
		
		// Wait a bit to ensure process is running
		time.Sleep(100 * time.Millisecond)

		// Now test the stop tool with SIGKILL
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		// Set up request arguments with kill=true
		argsJSON := fmt.Sprintf(`{"pid": %d, "kill": true}`, pid)
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify result content
		assert.Len(t, result.Content, 1)
		assert.IsType(t, &mcp.TextContent{}, result.Content[0])
		
		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Signal SIGKILL sent to process")
		assert.Contains(t, textContent.Text, fmt.Sprintf("%d", pid))

		// Verify data structure
		resultData, ok := data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, pid, resultData["pid"])
		assert.Equal(t, "SIGKILL", resultData["signal"])
		assert.Equal(t, "signal_sent", resultData["status"])

		// Wait for process to actually terminate
		cmd.Wait()
	})

	t.Run("non-existent process", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		// Use a very high PID that's unlikely to exist
		argsJSON := `{"pid": 99999, "kill": false}`
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
		// The error message may vary, but should contain an error about the process
		assert.Contains(t, textContent.Text, "Error sending signal to process 99999")
	})

	t.Run("invalid PID (zero)", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		argsJSON := `{"pid": 0, "kill": false}`
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
		assert.Contains(t, textContent.Text, "PID is required")
	})

	t.Run("process termination is non-blocking", func(t *testing.T) {
		// Start a long-running process
		var cmd *exec.Cmd
		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "ping", "127.0.0.1", "-n", "10"}
		} else {
			command = "sleep"
			args = []string{"10"}
		}
		
		cmd = exec.Command(command, args...)
		err := cmd.Start()
		require.NoError(t, err)
		
		pid := cmd.Process.Pid
		
		// Wait a bit to ensure process is running
		time.Sleep(100 * time.Millisecond)

		// Test that the stop function returns quickly
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		argsJSON := fmt.Sprintf(`{"pid": %d, "kill": false}`, pid)
		req.Params.Arguments = json.RawMessage(argsJSON)

		start := time.Now()
		result, data, err := tool.Handle(ctx, req)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify the function returns quickly (non-blocking)
		assert.Less(t, elapsed, 1*time.Second, "Function should return immediately after sending signal")

		// Clean up
		cmd.Wait()
	})
}

func TestStopProcessToolIntegration(t *testing.T) {
	tool := NewStopProcessTool()

	t.Run("complete workflow with exec and stop tools", func(t *testing.T) {
		// This test demonstrates the complete workflow that would be used by an agent:
		// 1. Start a process using exec tool (simulated)
		// 2. Get the PID from the response
		// 3. Stop the process using stop tool with that PID
		
		// Step 1: Start a process (simulating what exec_process tool would do)
		var cmd *exec.Cmd
		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "ping", "127.0.0.1", "-n", "3"}
		} else {
			command = "sleep"
			args = []string{"3"}
		}
		
		cmd = exec.Command(command, args...)
		err := cmd.Start()
		require.NoError(t, err)
		
		pid := cmd.Process.Pid
		t.Logf("Started process with PID: %d", pid)

		// Step 2: Wait a bit to ensure process is running
		time.Sleep(100 * time.Millisecond)

		// Step 3: Stop the process using stop_process tool with the PID
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: tool.Name(),
			},
		}

		argsJSON := fmt.Sprintf(`{"pid": %d, "kill": false}`, pid)
		req.Params.Arguments = json.RawMessage(argsJSON)

		result, data, err := tool.Handle(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify the stop was successful
		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Signal SIGTERM sent to process")
		assert.Contains(t, textContent.Text, fmt.Sprintf("%d", pid))

		// Verify the data structure
		resultData, ok := data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, pid, resultData["pid"])
		assert.Equal(t, "SIGTERM", resultData["signal"])
		assert.Equal(t, "signal_sent", resultData["status"])

		t.Logf("Successfully stopped process with PID: %d", pid)

		// Step 4: Wait for process to actually terminate
		cmd.Wait()
	})
}