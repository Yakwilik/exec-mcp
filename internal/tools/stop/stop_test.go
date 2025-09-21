package stop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestStopProcessTool(t *testing.T) {
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
		
		// Verify process is running
		process, err := os.FindProcess(pid)
		require.NoError(t, err)
		require.NotNil(t, process)

		// Now test the stop tool
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "stop_process",
			},
		}

		stopArgs := Args{
			PID:  pid,
			Kill: false, // Use SIGTERM
		}

		result, data, err := StopProcessTool(ctx, req, stopArgs)

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
				Name: "stop_process",
			},
		}

		stopArgs := Args{
			PID:  pid,
			Kill: true, // Use SIGKILL
		}

		result, data, err := StopProcessTool(ctx, req, stopArgs)

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
				Name: "stop_process",
			},
		}

		// Use a very high PID that's unlikely to exist
		stopArgs := Args{
			PID:  99999,
			Kill: false,
		}

		result, data, err := StopProcessTool(ctx, req, stopArgs)

		// Should return an error
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, data)

		// Verify error content
		assert.Len(t, result.Content, 1)
		assert.IsType(t, &mcp.TextContent{}, result.Content[0])
		
		textContent := result.Content[0].(*mcp.TextContent)
		// The exact error message may vary, but should contain an error about the process
		assert.Contains(t, textContent.Text, "Error sending signal to process 99999")
	})

	t.Run("invalid PID (zero)", func(t *testing.T) {
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "stop_process",
			},
		}

		stopArgs := Args{
			PID:  0,
			Kill: false,
		}

		result, data, err := StopProcessTool(ctx, req, stopArgs)

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
				Name: "stop_process",
			},
		}

		stopArgs := Args{
			PID:  pid,
			Kill: false,
		}

		start := time.Now()
		result, data, err := StopProcessTool(ctx, req, stopArgs)
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
	t.Run("start and stop process workflow", func(t *testing.T) {
		// This test simulates the typical workflow: start a process, then stop it
		
		// Step 1: Start a process
		var cmd *exec.Cmd
		var command string
		var args []string
		
		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "ping", "127.0.0.1", "-n", "5"}
		} else {
			command = "sleep"
			args = []string{"5"}
		}
		
		cmd = exec.Command(command, args...)
		err := cmd.Start()
		require.NoError(t, err)
		
		pid := cmd.Process.Pid
		
		// Verify process is running
		process, err := os.FindProcess(pid)
		require.NoError(t, err)
		require.NotNil(t, process)

		// Wait a bit to ensure process is running
		time.Sleep(100 * time.Millisecond)

		// Step 2: Stop the process using our tool
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "stop_process",
			},
		}

		stopArgs := Args{
			PID:  pid,
			Kill: false,
		}

		result, data, err := StopProcessTool(ctx, req, stopArgs)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, data)

		// Verify the stop was successful
		textContent := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, textContent.Text, "Signal SIGTERM sent to process")
		assert.Contains(t, textContent.Text, fmt.Sprintf("%d", pid))

		// Step 3: Wait for process to actually terminate
		cmd.Wait()
	})

	t.Run("multiple process termination", func(t *testing.T) {
		// Start multiple processes
		numProcesses := 3
		cmds := make([]*exec.Cmd, numProcesses)
		pids := make([]int, numProcesses)
		
		for i := 0; i < numProcesses; i++ {
			var command string
			var args []string
			
			if runtime.GOOS == "windows" {
				command = "cmd"
				args = []string{"/c", "ping", "127.0.0.1", "-n", "5"}
			} else {
				command = "sleep"
				args = []string{"5"}
			}
			
			cmd := exec.Command(command, args...)
			err := cmd.Start()
			require.NoError(t, err)
			
			cmds[i] = cmd
			pids[i] = cmd.Process.Pid
		}

		// Wait a bit to ensure processes are running
		time.Sleep(100 * time.Millisecond)

		// Stop all processes
		ctx := context.Background()
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Name: "stop_process",
			},
		}

		for i, pid := range pids {
			stopArgs := Args{
				PID:  pid,
				Kill: false,
			}

			result, data, err := StopProcessTool(ctx, req, stopArgs)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, data)

			// Verify each stop was successful
			textContent := result.Content[0].(*mcp.TextContent)
			assert.Contains(t, textContent.Text, "Signal SIGTERM sent to process")
			assert.Contains(t, textContent.Text, fmt.Sprintf("%d", pid))

			// Clean up
			cmds[i].Wait()
		}
	})
}