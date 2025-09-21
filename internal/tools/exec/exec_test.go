package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestScript creates a temporary script for testing
func createTestScript(t *testing.T, script string) string {
	dir := t.TempDir()
	
	var scriptFile string
	var scriptContent string
	
	if runtime.GOOS == "windows" {
		scriptFile = filepath.Join(dir, "test.bat")
		scriptContent = script
	} else {
		scriptFile = filepath.Join(dir, "test.sh")
		scriptContent = "#!/bin/bash\n" + script
	}
	
	err := os.WriteFile(scriptFile, []byte(scriptContent), 0755)
	require.NoError(t, err)
	
	return scriptFile
}

func TestExecProcessTool(t *testing.T) {
	t.Run("successful process execution", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "exec_process",
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

		execArgs := Args{
			Command: command,
			Args:    args,
		}

		result, data, err := ExecProcessTool(ctx, req, execArgs)

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

	t.Run("process execution with custom directory", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "exec_process",
			},
		}

		tempDir := t.TempDir()
		
		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "cd"}
		} else {
			command = "pwd"
			args = []string{}
		}

		execArgs := Args{
			Command: command,
			Args:    args,
			Dir:     tempDir,
		}

		result, data, err := ExecProcessTool(ctx, req, execArgs)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		processInfo, ok := data.(ProcessInfo)
		require.True(t, ok)
		assert.Greater(t, processInfo.PID, 0)
		assert.Equal(t, tempDir, execArgs.Dir)
	})

	t.Run("process execution with environment variables", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "exec_process",
			},
		}

		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "echo", "%TEST_VAR%"}
		} else {
			command = "echo"
			args = []string{"$TEST_VAR"}
		}

		execArgs := Args{
			Command: command,
			Args:    args,
			Env:     []string{"TEST_VAR=hello_world"},
		}

		result, data, err := ExecProcessTool(ctx, req, execArgs)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		processInfo, ok := data.(ProcessInfo)
		require.True(t, ok)
		assert.Greater(t, processInfo.PID, 0)
		assert.Equal(t, []string{"TEST_VAR=hello_world"}, execArgs.Env)
	})

	t.Run("long running process (detached)", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "exec_process",
			},
		}

		// Create a script that sleeps for 10 seconds
		script := createTestScript(t, "sleep 10")
		
		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = script
			args = []string{}
		} else {
			command = script
			args = []string{}
		}

		execArgs := Args{
			Command: command,
			Args:    args,
		}

		start := time.Now()
		result, data, err := ExecProcessTool(ctx, req, execArgs)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify the function returns quickly (non-blocking)
		assert.Less(t, elapsed, 1*time.Second, "Function should return immediately, not wait for process completion")

		processInfo, ok := data.(ProcessInfo)
		require.True(t, ok)
		assert.Greater(t, processInfo.PID, 0)

		// Verify process is actually running
		process, err := os.FindProcess(processInfo.PID)
		require.NoError(t, err)
		
		// Send SIGTERM to clean up
		process.Signal(syscall.SIGTERM)
	})

	t.Run("invalid command", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "exec_process",
			},
		}

		execArgs := Args{
			Command: "nonexistent_command_12345",
			Args:    []string{},
		}

		result, data, err := ExecProcessTool(ctx, req, execArgs)

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

	t.Run("missing command", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "exec_process",
			},
		}

		execArgs := Args{
			Command: "",
			Args:    []string{},
		}

		result, data, err := ExecProcessTool(ctx, req, execArgs)

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

func TestConcurrentProcessExecution(t *testing.T) {
	// Test that multiple processes can be started concurrently
	ctx := context.Background()
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "exec_process",
		},
	}

	numProcesses := 5
	results := make(chan ProcessInfo, numProcesses)
	errors := make(chan error, numProcesses)

	for i := 0; i < numProcesses; i++ {
		go func(index int) {
			var command string
			var args []string
			
			if runtime.GOOS == "windows" {
				command = "cmd"
				args = []string{"/c", "echo", fmt.Sprintf("process-%d", index)}
			} else {
				command = "echo"
				args = []string{fmt.Sprintf("process-%d", index)}
			}

			execArgs := Args{
				Command: command,
				Args:    args,
			}

			_, data, err := ExecProcessTool(ctx, req, execArgs)
			if err != nil {
				errors <- err
				return
			}

			processInfo, ok := data.(ProcessInfo)
			if !ok {
				errors <- fmt.Errorf("expected ProcessInfo type")
				return
			}

			results <- processInfo
		}(i)
	}

	// Collect results
	collectedPIDs := make(map[int]bool)
	for i := 0; i < numProcesses; i++ {
		select {
		case processInfo := <-results:
			assert.Greater(t, processInfo.PID, 0)
			assert.False(t, collectedPIDs[processInfo.PID], "PID should be unique")
			collectedPIDs[processInfo.PID] = true
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent processes")
		}
	}

	assert.Len(t, collectedPIDs, numProcesses, "Should have unique PIDs for all processes")
}