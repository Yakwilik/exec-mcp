# Exec MCP Server

A Model Context Protocol (MCP) server that provides tools for executing and managing processes. This server allows agents to execute processes and stop them by PID, making process management easier than using terminal commands.

## Features

- **exec_process**: Execute a process and return its PID for later management
- **stop_process**: Stop a process by its PID using SIGTERM or SIGKILL

## Building

```bash
go build -o exec-mcp ./cmd/exec-mcp
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

## Benefits

This MCP server enables agents to:
- Execute processes and get their PIDs for precise management
- Stop processes without needing to search for them using terminal commands
- Manage multiple processes concurrently with their exact process IDs
- Use proper signal handling (SIGTERM/SIGKILL) for clean process termination

Without this MCP, agents typically need to use terminal commands like `ps`, `grep`, and `kill` to find and manage processes, which is less reliable and more complex.