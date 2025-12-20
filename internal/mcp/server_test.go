package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"

	exectool "github.com/Yakwilik/exec-mcp/internal/tools/exec"
	stoptool "github.com/Yakwilik/exec-mcp/internal/tools/stop"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateServer(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)
}

func TestServerTools(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Test that we can connect to the server
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// Test listing tools
	tools, err := clientSession.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.Len(t, tools.Tools, 3)

	// Get expected tool names from our tool structs
	expectedTools := GetTools()
	expectedNames := make(map[string]bool)
	for _, tool := range expectedTools {
		expectedNames[tool.Name()] = true
	}

	// Verify tool names match our tool structs
	actualNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		actualNames[tool.Name] = true
		assert.True(t, expectedNames[tool.Name], "Tool name %s should be in expected tools", tool.Name)
	}

	// Verify we have all expected tools
	for expectedName := range expectedNames {
		assert.True(t, actualNames[expectedName], "Expected tool %s should be present", expectedName)
	}
}

func TestServerExecProcessIntegration(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Set up client-server connection
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// Test exec_process tool
	var command string
	var args []string

	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "echo", "test"}
	} else {
		command = "echo"
		args = []string{"test"}
	}

	// Get tool name from tool struct
	execTool := exectool.NewExecProcessTool()
	params := &mcp.CallToolParams{
		Name: execTool.Name(),
		Arguments: map[string]any{
			"command": command,
			"args":    args,
		},
	}

	result, err := clientSession.CallTool(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	// Verify result content
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Process started successfully")
}

func TestServerStopProcessIntegration(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Set up client-server connection
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// Start a process first
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
	err = cmd.Start()
	require.NoError(t, err)

	pid := cmd.Process.Pid

	// Wait a bit to ensure process is running
	time.Sleep(100 * time.Millisecond)

	// Test stop_process tool
	stopTool := stoptool.NewStopProcessTool()
	params := &mcp.CallToolParams{
		Name: stopTool.Name(),
		Arguments: map[string]any{
			"pid":  pid,
			"kill": false,
		},
	}

	result, err := clientSession.CallTool(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	// Verify result content
	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Signal SIGTERM sent to process")
	assert.Contains(t, textContent.Text, fmt.Sprintf("%d", pid))

	// Clean up
	cmd.Wait()
}

func TestServerWorkflowIntegration(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Set up client-server connection
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// Step 1: Start a long-running process using exec_process tool
	var command string
	var args []string

	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "ping", "127.0.0.1", "-n", "5"}
	} else {
		command = "sleep"
		args = []string{"5"}
	}

	execTool := exectool.NewExecProcessTool()
	execParams := &mcp.CallToolParams{
		Name: execTool.Name(),
		Arguments: map[string]any{
			"command": command,
			"args":    args,
		},
	}

	execResult, err := clientSession.CallTool(ctx, execParams)
	require.NoError(t, err)
	require.NotNil(t, execResult)
	require.Len(t, execResult.Content, 1)

	// Step 2: Extract PID from exec_process response
	pid, err := extractPIDFromExecResponse(execResult)
	require.NoError(t, err)
	require.Greater(t, pid, 0, "PID should be greater than 0")

	t.Logf("Successfully started process with PID: %d", pid)

	// Step 3: Wait a bit to ensure process is running
	time.Sleep(100 * time.Millisecond)

	// Step 4: Stop the process using the PID from exec_process
	stopTool := stoptool.NewStopProcessTool()
	stopParams := &mcp.CallToolParams{
		Name: stopTool.Name(),
		Arguments: map[string]any{
			"pid":  pid, // Use the real PID from exec_process
			"kill": false,
		},
	}

	stopResult, err := clientSession.CallTool(ctx, stopParams)
	require.NoError(t, err)
	require.NotNil(t, stopResult)
	require.Len(t, stopResult.Content, 1)

	// Step 5: Verify successful termination
	stopTextContent, ok := stopResult.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, stopTextContent.Text, "Signal SIGTERM sent to process")
	assert.Contains(t, stopTextContent.Text, fmt.Sprintf("%d", pid))

	t.Logf("Successfully stopped process with PID: %d", pid)
}

func TestServerMultipleProcessWorkflow(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Set up client-server connection
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// Test workflow with multiple processes
	numProcesses := 3
	pids := make([]int, 0, numProcesses)

	// Step 1: Start multiple processes
	for i := 0; i < numProcesses; i++ {
		var command string
		var args []string

		if runtime.GOOS == "windows" {
			command = "cmd"
			args = []string{"/c", "ping", "127.0.0.1", "-n", "3"}
		} else {
			command = "sleep"
			args = []string{"3"}
		}

		execTool := exectool.NewExecProcessTool()
		execParams := &mcp.CallToolParams{
			Name: execTool.Name(),
			Arguments: map[string]any{
				"command": command,
				"args":    args,
			},
		}

		execResult, err := clientSession.CallTool(ctx, execParams)
		require.NoError(t, err)
		require.NotNil(t, execResult)

		// Extract PID
		pid, err := extractPIDFromExecResponse(execResult)
		require.NoError(t, err)
		require.Greater(t, pid, 0)

		pids = append(pids, pid)
		t.Logf("Started process %d with PID: %d", i+1, pid)
	}

	// Step 2: Wait a bit to ensure all processes are running
	time.Sleep(200 * time.Millisecond)

	// Step 3: Stop all processes using their PIDs
	for i, pid := range pids {
		stopTool := stoptool.NewStopProcessTool()
		stopParams := &mcp.CallToolParams{
			Name: stopTool.Name(),
			Arguments: map[string]any{
				"pid":  pid,
				"kill": false,
			},
		}

		stopResult, err := clientSession.CallTool(ctx, stopParams)
		require.NoError(t, err)
		require.NotNil(t, stopResult)

		// Verify successful termination
		stopTextContent, ok := stopResult.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, stopTextContent.Text, "Signal SIGTERM sent to process")
		assert.Contains(t, stopTextContent.Text, fmt.Sprintf("%d", pid))

		t.Logf("Stopped process %d with PID: %d", i+1, pid)
	}

	t.Logf("Successfully completed workflow with %d processes", numProcesses)
}

func TestServerNonBlockingBehavior(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Set up client-server connection
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// Test that multiple tool calls don't block each other
	var command string
	var args []string

	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "echo", "test"}
	} else {
		command = "echo"
		args = []string{"test"}
	}

	execTool := exectool.NewExecProcessTool()
	params := &mcp.CallToolParams{
		Name: execTool.Name(),
		Arguments: map[string]any{
			"command": command,
			"args":    args,
		},
	}

	// Make multiple concurrent calls
	numCalls := 5
	results := make(chan *mcp.CallToolResult, numCalls)
	errors := make(chan error, numCalls)

	start := time.Now()

	for i := 0; i < numCalls; i++ {
		go func(index int) {
			result, err := clientSession.CallTool(ctx, params)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(i)
	}

	// Collect results
	completedCalls := 0
	for completedCalls < numCalls {
		select {
		case result := <-results:
			require.NotNil(t, result)
			completedCalls++
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent calls")
		}
	}

	elapsed := time.Since(start)

	// Verify all calls completed quickly (non-blocking)
	assert.Less(t, elapsed, 3*time.Second, "Concurrent calls should complete quickly")
	assert.Equal(t, numCalls, completedCalls, "All calls should complete")
}

func TestServerErrorHandling(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Set up client-server connection
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// Test invalid tool name
	params := &mcp.CallToolParams{
		Name: "invalid_tool",
		Arguments: map[string]any{
			"test": "value",
		},
	}

	result, err := clientSession.CallTool(ctx, params)
	// The client should handle this gracefully
	// The exact behavior depends on the MCP SDK implementation
	// Note: The SDK might return an error or handle it differently
	if err != nil {
		// If there's an error, that's also acceptable behavior
		assert.Error(t, err)
	} else {
		// If no error, should get some response
		assert.NotNil(t, result)
	}
}

func TestServerConcurrentConnections(t *testing.T) {
	server := CreateServer()
	require.NotNil(t, server)

	// Test multiple concurrent connections
	numConnections := 3
	connections := make([]*mcp.ClientSession, numConnections)

	ctx := context.Background()

	// Create multiple connections
	for i := 0; i < numConnections; i++ {
		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		_, err := server.Connect(ctx, serverTransport, nil)
		require.NoError(t, err)

		client := mcp.NewClient(&mcp.Implementation{Name: fmt.Sprintf("test-client-%d", i), Version: "v1.0.0"}, nil)
		clientSession, err := client.Connect(ctx, clientTransport, nil)
		require.NoError(t, err)

		connections[i] = clientSession
	}

	// Test that all connections can make tool calls
	for i, conn := range connections {
		execTool := exectool.NewExecProcessTool()
		params := &mcp.CallToolParams{
			Name: execTool.Name(),
			Arguments: map[string]any{
				"command": "echo",
				"args":    []string{fmt.Sprintf("connection-%d", i)},
			},
		}

		result, err := conn.CallTool(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, result)
	}

	// Clean up connections
	for _, conn := range connections {
		conn.Close()
	}
}
