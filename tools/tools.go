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
	"gopherclaw/rag"

	"github.com/tmc/langchaingo/llms"
)

var mcpManager *mcp.Manager
var ragModule *rag.RAG

func SetMCPManager(m *mcp.Manager) {
	mcpManager = m
}

func SetRAG(r *rag.RAG) {
	ragModule = r
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
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "search_knowledge",
				Description: "Search the knowledge base for relevant information. Use this when you need facts, documentation, or context about the project.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "The search query to find relevant knowledge",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "store_knowledge",
				Description: "Store text in the knowledge base for future retrieval. Use this to save important information you've discovered.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "The text content to store in the knowledge base",
						},
						"source": map[string]any{
							"type":        "string",
							"description": "A label describing where this info came from (e.g. 'main.go', 'user request')",
						},
					},
					"required": []string{"content", "source"},
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
	case "search_knowledge":
		return searchKnowledge(args["query"])
	case "store_knowledge":
		return storeKnowledge(args["content"], args["source"])
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

func storeKnowledge(content, source string) string {
	if ragModule == nil {
		return "error: knowledge base not configured"
	}
	docs := rag.ChunkAndWrap(content, source)
	if err := ragModule.Ingest(context.Background(), docs); err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return fmt.Sprintf("Stored %d chunks from %q in knowledge base.", len(docs), source)
}

func searchKnowledge(query string) string {
	if ragModule == nil {
		return "error: knowledge base not configured"
	}
	result := ragModule.Retrieve(context.Background(), query, 3)
	if result == "" {
		return "No relevant results found."
	}
	return result
}
