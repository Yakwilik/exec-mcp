# Exec MCP Server

A Model Context Protocol (MCP) server that provides tools for executing and managing processes. This server allows agents to execute processes and stop them by PID, making process management easier than using terminal commands.

## Features

- **exec_process**: Execute a process and return its PID for later management
- **stop_process**: Stop a process by its PID using SIGTERM or SIGKILL
- **Non-blocking**: Processes run detached and don't block the MCP server
- **Concurrent**: Multiple processes can be managed simultaneously

## Building

```bash
go build -o exec-mcp ./cmd/exec-mcp
```

## Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test ./... -v

# Run specific tool tests
go test ./internal/tools/exec/ -v
go test ./internal/tools/stop/ -v

# Run integration tests
go test ./internal/mcp/ -v
```

## Usage

The server uses stdio transport for communication:

```bash
./exec-mcp
```

## Tools

### exec_process

Execute a process and return its PID.

**Parameters:**
- `command` (string, required): The command to execute
- `args` (array of strings, optional): Command line arguments
- `dir` (string, optional): Working directory for the process
- `env` (array of strings, optional): Environment variables (format: KEY=VALUE)

**Example:**
```json
{
  "command": "ls",
  "args": ["-l", "/tmp"],
  "dir": "/home/user"
}
```

**Response:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "Process started successfully:\n{\n  \"pid\": 12345,\n  \"command\": \"ls\",\n  \"args\": [\"-l\", \"/tmp\"],\n  \"start_time\": \"2024-01-01T12:00:00Z\",\n  \"status\": \"running\"\n}"
    }
  ],
  "structuredContent": {
    "pid": 12345,
    "command": "ls",
    "args": ["-l", "/tmp"],
    "start_time": "2024-01-01T12:00:00Z",
    "status": "running"
  }
}
```

**Note:** The `structuredContent` field contains the structured data that can be directly accessed without parsing JSON text.

### stop_process

Stop a process by its PID using SIGTERM or SIGKILL.

**Parameters:**
- `pid` (integer, required): The process ID to stop
- `kill` (boolean, optional): If true, use SIGKILL instead of SIGTERM (force kill)

**Example:**
```json
{
  "pid": 12345,
  "kill": false
}
```

**Response:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "Signal SIGTERM sent to process 12345"
    }
  ],
  "structuredContent": {
    "pid": 12345,
    "signal": "SIGTERM",
    "status": "signal_sent"
  }
}
```

## Using Structured Data

The MCP server returns both human-readable text and structured JSON data. For programmatic access, use the `structuredContent` field:

```go
// Example: Extract PID from exec_process response
result, err := clientSession.CallTool(ctx, execParams)
if err != nil {
    log.Fatal(err)
}

// Access structured data directly
if processInfoMap, ok := result.StructuredContent.(map[string]interface{}); ok {
    if pidFloat, exists := processInfoMap["pid"]; exists {
        if pid, ok := pidFloat.(float64); ok {
            processID := int(pid)
            // Use processID for stop_process
        }
    }
}
```

## Architecture

The project is organized into separate packages for maintainability:

- `./cmd/exec-mcp/main.go`: Main entry point
- `./internal/mcp/server.go`: MCP server setup and configuration
- `./internal/tools/tool.go`: Tool interface definition
- `./internal/tools/exec/`: Process execution tool (struct-based)
- `./internal/tools/stop/`: Process termination tool (struct-based)
- `./internal/mcp/test_helpers.go`: Helper functions for extracting structured data

### Tool Architecture

Each tool is implemented as a struct that implements the `Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Handle(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error)
}
```

This approach provides:
- **No hardcoded tool names**: Names are retrieved from tool structs
- **Self-documenting**: Each tool provides its own name and description
- **Type safety**: Interface ensures consistent tool implementation
- **Easy testing**: Tools can be tested independently

## Benefits

This MCP server enables agents to:
- Execute processes and get their PIDs for precise management
- Stop processes without needing to search for them using terminal commands
- Manage multiple processes concurrently with their exact process IDs
- Use proper signal handling (SIGTERM/SIGKILL) for clean process termination
- Run processes in detached mode without blocking the MCP server

Without this MCP, agents typically need to use terminal commands like `ps`, `grep`, and `kill` to find and manage processes, which is less reliable and more complex.