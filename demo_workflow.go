package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	servermcp "exec-mcp/internal/mcp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func extractPIDFromResponse(result *mcp.CallToolResult) (int, error) {
	if len(result.Content) == 0 {
		return 0, fmt.Errorf("no content in response")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return 0, fmt.Errorf("expected text content")
	}

	// Simple regex to extract PID from JSON
	text := textContent.Text
	start := -1
	for i := 0; i < len(text)-3; i++ {
		if text[i:i+5] == `"pid":` {
			start = i + 5
			break
		}
	}
	
	if start == -1 {
		return 0, fmt.Errorf("PID not found in response")
	}

	// Find the number
	for start < len(text) && (text[start] == ' ' || text[start] == '\t' || text[start] == '\n') {
		start++
	}
	
	end := start
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
	}
	
	if start == end {
		return 0, fmt.Errorf("no PID number found")
	}

	var pid int
	for i := start; i < end; i++ {
		pid = pid*10 + int(text[i]-'0')
	}
	
	return pid, nil
}

func main() {
	fmt.Println("=== MCP Exec Server Workflow Demo ===")
	fmt.Println("This demo shows the complete workflow:")
	fmt.Println("1. Start process using exec_process tool")
	fmt.Println("2. Extract PID from response")
	fmt.Println("3. Stop process using stop_process tool with that PID")
	fmt.Println()

	// Create the MCP server
	server := servermcp.CreateServer()

	// Set up client-server connection
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "demo-client", Version: "v1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer clientSession.Close()

	// Demo 1: Single process workflow
	fmt.Println("--- Demo 1: Single Process Workflow ---")
	
	var command string
	var args []string
	
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "ping", "127.0.0.1", "-n", "3"}
	} else {
		command = "sleep"
		args = []string{"3"}
	}

	// Step 1: Start process
	fmt.Printf("Starting process: %s %v\n", command, args)
	execParams := &mcp.CallToolParams{
		Name: "exec_process",
		Arguments: map[string]any{
			"command": command,
			"args":    args,
		},
	}

	execResult, err := clientSession.CallTool(ctx, execParams)
	if err != nil {
		log.Fatal("Failed to start process:", err)
	}

	// Step 2: Extract PID
	pid, err := extractPIDFromResponse(execResult)
	if err != nil {
		log.Fatal("Failed to extract PID:", err)
	}
	fmt.Printf("✅ Process started with PID: %d\n", pid)

	// Step 3: Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Step 4: Stop process
	fmt.Printf("Stopping process with PID: %d\n", pid)
	stopParams := &mcp.CallToolParams{
		Name: "stop_process",
		Arguments: map[string]any{
			"pid":  pid,
			"kill": false,
		},
	}

	stopResult, err := clientSession.CallTool(ctx, stopParams)
	if err != nil {
		log.Fatal("Failed to stop process:", err)
	}

	stopTextContent := stopResult.Content[0].(*mcp.TextContent)
	fmt.Printf("✅ %s\n", stopTextContent.Text)
	fmt.Println()

	// Demo 2: Multiple processes workflow
	fmt.Println("--- Demo 2: Multiple Processes Workflow ---")
	
	numProcesses := 3
	pids := make([]int, 0, numProcesses)

	// Start multiple processes
	for i := 0; i < numProcesses; i++ {
		fmt.Printf("Starting process %d...\n", i+1)
		
		execResult, err := clientSession.CallTool(ctx, execParams)
		if err != nil {
			log.Fatal("Failed to start process:", err)
		}

		pid, err := extractPIDFromResponse(execResult)
		if err != nil {
			log.Fatal("Failed to extract PID:", err)
		}
		
		pids = append(pids, pid)
		fmt.Printf("✅ Process %d started with PID: %d\n", i+1, pid)
	}

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Stop all processes
	fmt.Println("\nStopping all processes...")
	for i, pid := range pids {
		fmt.Printf("Stopping process %d (PID: %d)...\n", i+1, pid)
		
		stopParams := &mcp.CallToolParams{
			Name: "stop_process",
			Arguments: map[string]any{
				"pid":  pid,
				"kill": false,
			},
		}

		stopResult, err := clientSession.CallTool(ctx, stopParams)
		if err != nil {
			log.Fatal("Failed to stop process:", err)
		}

		stopTextContent := stopResult.Content[0].(*mcp.TextContent)
		fmt.Printf("✅ %s\n", stopTextContent.Text)
	}

	fmt.Println()
	fmt.Println("=== Demo Complete ===")
	fmt.Println("This demonstrates how agents can use the MCP server to:")
	fmt.Println("- Start processes and get their PIDs")
	fmt.Println("- Stop processes using their exact PIDs")
	fmt.Println("- Manage multiple processes concurrently")
	fmt.Println("- Avoid the need for terminal commands like 'ps' and 'kill'")
}