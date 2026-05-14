package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"

	exectool "github.com/Yakwilik/exec-mcp/internal/tools/exec"
	runtool "github.com/Yakwilik/exec-mcp/internal/tools/run"
	stoptool "github.com/Yakwilik/exec-mcp/internal/tools/stop"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer creates a server for testing and fails the test if it cannot be created.
func newTestServer(t *testing.T) *mcp.Server {
	t.Helper()
	server, err := CreateServer()
	require.NoError(t, err)
	require.NotNil(t, server)
	return server
}

// newTestClientSession connects a test client to the server and returns the session.
func newTestClientSession(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { clientSession.Close() })

	return clientSession
}

func TestCreateServer(t *testing.T) {
	server, err := CreateServer()
	require.NoError(t, err)
	require.NotNil(t, server)
}

func TestServerTools(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

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
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

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
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      execTool.Name(),
		Arguments: map[string]any{"command": command, "args": args},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, `"status":"running"`)
}

func TestServerStopProcessIntegration(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	// Start a process to stop.
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
	pid := cmd.Process.Pid
	time.Sleep(100 * time.Millisecond)

	stopTool := stoptool.NewStopProcessTool()
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      stopTool.Name(),
		Arguments: map[string]any{"pid": pid, "kill": false},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, `"signal":"SIGTERM"`)
	assert.Contains(t, textContent.Text, fmt.Sprintf("%d", pid))
	cmd.Wait()
}

func TestServerRunCommandSuccess(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	var command string
	var args []string
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "echo", "hello"}
	} else {
		command = "echo"
		args = []string{"hello"}
	}

	runTool := runtool.NewRunCommandTool()
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      runTool.Name(),
		Arguments: map[string]any{"command": command, "args": args},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsError, "successful command should not set IsError")

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, `"success":true`)
	assert.Contains(t, textContent.Text, `"exitCode":0`)
	assert.Contains(t, textContent.Text, "hello")
}

func TestServerRunCommandNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell exit test on Windows")
	}

	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	runTool := runtool.NewRunCommandTool()
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      runTool.Name(),
		Arguments: map[string]any{"command": "sh", "args": []string{"-c", "exit 42"}},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	// Non-zero exit is returned as a structured response (not a tool error).
	require.False(t, result.IsError, "non-zero exit should return structured response, not IsError")

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, `"success":false`)
	assert.Contains(t, textContent.Text, `"exitCode":42`)
}

func TestServerRunCommandInvalidCommand(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	runTool := runtool.NewRunCommandTool()
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      runTool.Name(),
		Arguments: map[string]any{"command": "this_command_definitely_does_not_exist_12345"},
	})
	require.NoError(t, err) // MCP-level call succeeds; handler returns an error result
	require.NotNil(t, result)
	assert.True(t, result.IsError, "invalid command should set IsError")
}

func TestServerRunCommandTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping timeout test on Windows")
	}

	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	runTool := runtool.NewRunCommandTool()
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: runTool.Name(),
		Arguments: map[string]any{
			"command": "sleep",
			"args":    []string{"30"},
			"timeout": 1,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "timed-out command should set IsError")
}

func TestServerWorkflowIntegration(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	var command string
	var args []string
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "ping", "127.0.0.1", "-n", "5"}
	} else {
		command = "sleep"
		args = []string{"5"}
	}

	// Step 1: Start a long-running process.
	execTool := exectool.NewExecProcessTool()
	execResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      execTool.Name(),
		Arguments: map[string]any{"command": command, "args": args},
	})
	require.NoError(t, err)
	require.NotNil(t, execResult)
	require.Len(t, execResult.Content, 1)

	// Step 2: Extract PID from exec_process response.
	pid, err := extractPIDFromExecResponse(execResult)
	require.NoError(t, err)
	require.Greater(t, pid, 0, "PID should be greater than 0")
	t.Logf("Started process with PID: %d", pid)

	time.Sleep(100 * time.Millisecond)

	// Step 3: Stop the process.
	stopTool := stoptool.NewStopProcessTool()
	stopResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      stopTool.Name(),
		Arguments: map[string]any{"pid": pid, "kill": false},
	})
	require.NoError(t, err)
	require.NotNil(t, stopResult)
	require.Len(t, stopResult.Content, 1)

	stopTextContent, ok := stopResult.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, stopTextContent.Text, `"signal":"SIGTERM"`)
	assert.Contains(t, stopTextContent.Text, fmt.Sprintf("%d", pid))
	t.Logf("Stopped process with PID: %d", pid)
}

func TestServerMultipleProcessWorkflow(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	numProcesses := 3
	pids := make([]int, 0, numProcesses)

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
		execResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      execTool.Name(),
			Arguments: map[string]any{"command": command, "args": args},
		})
		require.NoError(t, err)

		pid, err := extractPIDFromExecResponse(execResult)
		require.NoError(t, err)
		require.Greater(t, pid, 0)

		pids = append(pids, pid)
		t.Logf("Started process %d with PID: %d", i+1, pid)
	}

	time.Sleep(200 * time.Millisecond)

	for i, pid := range pids {
		stopTool := stoptool.NewStopProcessTool()
		stopResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      stopTool.Name(),
			Arguments: map[string]any{"pid": pid, "kill": false},
		})
		require.NoError(t, err)
		require.NotNil(t, stopResult)

		stopTextContent, ok := stopResult.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, stopTextContent.Text, `"signal":"SIGTERM"`)
		assert.Contains(t, stopTextContent.Text, fmt.Sprintf("%d", pid))
		t.Logf("Stopped process %d with PID: %d", i+1, pid)
	}
}

func TestServerNonBlockingBehavior(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

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
		Name:      execTool.Name(),
		Arguments: map[string]any{"command": command, "args": args},
	}

	numCalls := 5
	results := make(chan *mcp.CallToolResult, numCalls)
	errs := make(chan error, numCalls)

	start := time.Now()
	for i := 0; i < numCalls; i++ {
		go func() {
			result, err := clientSession.CallTool(ctx, params)
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}

	completedCalls := 0
	for completedCalls < numCalls {
		select {
		case result := <-results:
			require.NotNil(t, result)
			completedCalls++
		case err := <-errs:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent calls")
		}
	}

	elapsed := time.Since(start)
	assert.Less(t, elapsed, 3*time.Second, "Concurrent calls should complete quickly")
	assert.Equal(t, numCalls, completedCalls, "All calls should complete")
}

func TestServerErrorHandling(t *testing.T) {
	server := newTestServer(t)
	clientSession := newTestClientSession(t, server)

	ctx := context.Background()

	params := &mcp.CallToolParams{
		Name:      "invalid_tool",
		Arguments: map[string]any{"test": "value"},
	}

	result, err := clientSession.CallTool(ctx, params)
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, result)
	}
}

func TestServerConcurrentConnections(t *testing.T) {
	server := newTestServer(t)

	numConnections := 3
	connections := make([]*mcp.ClientSession, numConnections)

	ctx := context.Background()

	for i := 0; i < numConnections; i++ {
		clientTransport, serverTransport := mcp.NewInMemoryTransports()

		_, err := server.Connect(ctx, serverTransport, nil)
		require.NoError(t, err)

		client := mcp.NewClient(&mcp.Implementation{Name: fmt.Sprintf("test-client-%d", i), Version: "v1.0.0"}, nil)
		clientSession, err := client.Connect(ctx, clientTransport, nil)
		require.NoError(t, err)

		connections[i] = clientSession
	}

	for i, conn := range connections {
		execTool := exectool.NewExecProcessTool()
		result, err := conn.CallTool(ctx, &mcp.CallToolParams{
			Name:      execTool.Name(),
			Arguments: map[string]any{"command": "echo", "args": []string{fmt.Sprintf("connection-%d", i)}},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	}

	for _, conn := range connections {
		conn.Close()
	}
}

