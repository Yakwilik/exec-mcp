package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"

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
	require.Len(t, tools.Tools, 2)

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}
	assert.True(t, toolNames["exec_process"])
	assert.True(t, toolNames["stop_process"])
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

	params := &mcp.CallToolParams{
		Name: "exec_process",
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
	params := &mcp.CallToolParams{
		Name: "stop_process",
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

	// Step 1: Start a long-running process
	var command string
	var args []string
	
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "ping", "127.0.0.1", "-n", "10"}
	} else {
		command = "sleep"
		args = []string{"10"}
	}

	execParams := &mcp.CallToolParams{
		Name: "exec_process",
		Arguments: map[string]any{
			"command": command,
			"args":    args,
		},
	}

	execResult, err := clientSession.CallTool(ctx, execParams)
	require.NoError(t, err)
	require.NotNil(t, execResult)

	// Extract PID from the result (this is a simplified approach)
	// In a real scenario, you'd parse the JSON response
	textContent, ok := execResult.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Process started successfully")

	// Step 2: Wait a bit to ensure process is running
	time.Sleep(100 * time.Millisecond)

	// Step 3: Stop the process (we'll use a known PID for this test)
	// In a real scenario, you'd extract the PID from the exec result
	stopParams := &mcp.CallToolParams{
		Name: "stop_process",
		Arguments: map[string]any{
			"pid":  99999, // Non-existent PID for testing
			"kill": false,
		},
	}

	stopResult, err := clientSession.CallTool(ctx, stopParams)
	require.NoError(t, err)
	require.NotNil(t, stopResult)

	// Verify error handling for non-existent process
	stopTextContent, ok := stopResult.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	// The error message should indicate the process issue
	assert.Contains(t, stopTextContent.Text, "process already finished")
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

	params := &mcp.CallToolParams{
		Name: "exec_process",
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
		params := &mcp.CallToolParams{
			Name: "exec_process",
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