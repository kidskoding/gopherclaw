package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopherclaw/mcp"

	"github.com/tmc/langchaingo/llms"
)

var mcpManager *mcp.Manager

func SetMCPManager(m *mcp.Manager) {
	mcpManager = m
}

func Definitions() []llms.Tool {
	local := []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "list_files",
				Description: "List files and directories at a given path",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Directory path to list (e.g. '.' for current directory)",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "read_file",
				Description: "Read the contents of a file",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the file to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "get_time",
				Description: "Get the current date and time",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
	}

	if mcpManager != nil {
		local = append(local, mcpManager.Tools()...)
	}

	return local
}

func Execute(name string, argsJSON string) string {
	if mcpManager != nil {
		if result, ok := mcpManager.Execute(context.Background(), name, argsJSON); ok {
			return result
		}
	}

	var args map[string]string
	json.Unmarshal([]byte(argsJSON), &args)

	switch name {
	case "list_files":
		return listFiles(args["path"])
	case "read_file":
		return readFile(args["path"])
	case "get_time":
		return getTime()
	default:
		return fmt.Sprintf("unknown tool: %s", name)
	}
}

func listFiles(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	return strings.Join(names, "\n")
}

func readFile(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	cwd, _ := os.Getwd()
	if !strings.HasPrefix(abs, cwd) {
		return "error: cannot read files outside the project directory"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	content := string(data)
	if len(content) > 4000 {
		content = content[:4000] + "\n... (truncated)"
	}
	return content
}

func getTime() string {
	return time.Now().Format("2006-01-02 15:04:05 MST")
}
