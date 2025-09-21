package mcp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// extractPIDFromExecResponse extracts the PID from an exec_process tool response
func extractPIDFromExecResponse(result *mcp.CallToolResult) (int, error) {
	// First, try to get PID from structured content
	if result.StructuredContent != nil {
		// The structured content should contain the ProcessInfo
		if processInfoMap, ok := result.StructuredContent.(map[string]interface{}); ok {
			if pidFloat, exists := processInfoMap["pid"]; exists {
				if pid, ok := pidFloat.(float64); ok {
					return int(pid), nil
				}
			}
		}
	}

	// Fallback: try to parse from text content
	if len(result.Content) == 0 {
		return 0, fmt.Errorf("no content in response")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return 0, fmt.Errorf("expected text content")
	}

	// Try to parse PID from the JSON response text
	// The response format is: "Process started successfully:\n{json}"
	lines := strings.Split(textContent.Text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			// This looks like JSON, try to parse it
			var processInfo struct {
				PID int `json:"pid"`
			}
			if err := json.Unmarshal([]byte(line), &processInfo); err == nil && processInfo.PID > 0 {
				return processInfo.PID, nil
			}
		}
	}

	// Fallback: try to extract PID using regex
	pidRegex := regexp.MustCompile(`"pid":\s*(\d+)`)
	matches := pidRegex.FindStringSubmatch(textContent.Text)
	if len(matches) > 1 {
		if pid, err := strconv.Atoi(matches[1]); err == nil {
			return pid, nil
		}
	}

	return 0, fmt.Errorf("could not extract PID from response: %s", textContent.Text)
}