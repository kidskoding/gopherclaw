package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/langchaingo/llms"
)

type serverConn struct {
	name    string
	session *gomcp.ClientSession
	prefix  string
	tools   []string
}

type Manager struct {
	servers []serverConn
}

func NewManager(ctx context.Context, configs []ServerConfig) (*Manager, error) {
	m := &Manager{}

	client := gomcp.NewClient(&gomcp.Implementation{
		Name:    "gopherclaw",
		Version: "1.0.0",
	}, nil)

	for _, cfg := range configs {
		cmd := exec.Command(cfg.Command, cfg.Args...)
		transport := &gomcp.CommandTransport{Command: cmd}

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			log.Printf("MCP: failed to connect to %s: %v", cfg.Name, err)
			continue
		}

		result, err := session.ListTools(ctx, nil)
		if err != nil {
			log.Printf("MCP: failed to list tools from %s: %v", cfg.Name, err)
			session.Close()
			continue
		}

		prefix := fmt.Sprintf("mcp_%s_", cfg.Name)
		var toolNames []string
		for _, t := range result.Tools {
			toolNames = append(toolNames, t.Name)
		}

		log.Printf("MCP: connected to %s (%d tools)", cfg.Name, len(toolNames))

		m.servers = append(m.servers, serverConn{
			name:    cfg.Name,
			session: session,
			prefix:  prefix,
			tools:   toolNames,
		})
	}

	return m, nil
}

func (m *Manager) Tools() []llms.Tool {
	var tools []llms.Tool

	for _, srv := range m.servers {
		result, err := srv.session.ListTools(context.Background(), nil)
		if err != nil {
			continue
		}
		for _, t := range result.Tools {
			tools = append(tools, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        srv.prefix + t.Name,
					Description: fmt.Sprintf("[MCP:%s] %s", srv.name, t.Description),
					Parameters:  t.InputSchema,
				},
			})
		}
	}

	return tools
}

func (m *Manager) Execute(ctx context.Context, name string, argsJSON string) (string, bool) {
	for _, srv := range m.servers {
		if strings.HasPrefix(name, srv.prefix) {
			realName := strings.TrimPrefix(name, srv.prefix)
			return m.callTool(ctx, srv.session, realName, argsJSON), true
		}
	}
	return "", false
}

func (m *Manager) callTool(ctx context.Context, session *gomcp.ClientSession, name string, argsJSON string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		args = map[string]any{}
	}

	result, err := session.CallTool(ctx, &gomcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return fmt.Sprintf("MCP error: %v", err)
	}

	if result.IsError {
		return fmt.Sprintf("MCP tool error: %s", extractText(result))
	}

	return extractText(result)
}

func extractText(result *gomcp.CallToolResult) string {
	var text string
	for _, c := range result.Content {
		if tc, ok := c.(*gomcp.TextContent); ok {
			text += tc.Text
		}
	}
	return text
}

func (m *Manager) Close() {
	for _, srv := range m.servers {
		srv.session.Close()
	}
}
