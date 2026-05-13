package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	ollama "github.com/ollama/ollama/api"
)

// ServerConn holds a live session to one MCP server subprocess along
// with the tools it advertised at startup. The agent treats every server
// identically; it does not care if the subprocess is in Go or Python.
type ServerConn struct {
	Name    string
	session *mcp.ClientSession
	// stored so agent loop can build full Ollama tool list once
	tools []*mcp.Tool
}

// discoverTools iterates over the server's tools list and caches them.
func (c *ServerConn) discoverTools(ctx context.Context) error {
	for tool, err := range c.session.Tools(ctx, nil) {
		if err != nil {
			return fmt.Errorf("list tools from %s: %w", c.Name, err)
		}
		c.tools = append(c.tools, tool)
	}
	return nil
}

// OllamaTools converts the server's MCP tool list into the format that
// the ollama API expects. The JSON schema for each tool's input is embedded
// verbatim; MCP and Ollama both use JSON schemas, so no translation is
// needed beyond the struct wrapping.
func (c *ServerConn) OllamaTools() []ollama.Tool {
	out := make([]ollama.Tool, 0, len(c.tools))
	for _, t := range c.tools {
		schema := toolsInputSchema(t)
		out = append(out, ollama.Tool{
			Type: "function",
			Function: ollama.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		})
	}
	return out
}

// Call dispatches a tool call to this server and returns the text content of the result. If the
// server signals is Error on the result, Call wraps the content as an error so the agent loop
// can surface it to the model.
func (c *ServerConn) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name: name, Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("call %s on %s: %w", name, c.Name, err)
	}
	text := extractText(result.Content)
	if result.IsError {
		return "", fmt.Errorf("tool %s error: %s", name, text)
	}
	return text, nil
}

// Close shuts down the session (and thus the subprocess).
func (c *ServerConn) Close() error {
	return c.session.Close()
}

// extractText joins all TextContent blocks from an MCP result into a single
// string. Non-text content (e.g. images) are ignored for now.
func extractText(content []mcp.Content) string {
	var out string
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out += tc.Text
		}
	}
	return out
}

// toolsInputSchema extracts a plain map[string]any from the MCP tool's
// InputSchema so that ollama.Tool can embed it. MCP stores the schema
// as json.RawMessage, so it is unmarshaled once here.
func toolsInputSchema(t *mcp.Tool) ollama.ToolFunctionParams {
	params := ollama.ToolFunctionParams{
		Type:       "object",
		Properties: map[string]ollama.ToolFunctionParamProperties{},
	}
	if t.InputSchema == nil {
		return params
	}

	// unmarshal raw JSON schema into a generic map so the 'properties'
	// and 'required' fields can be read without importing a schema library
	var raw struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	// FIXME: t.InputSchema can't be converted to bytes array
	if err := json.Unmarshal(t.InputSchema, &raw); err != nil {
		return params
	}

	for name, propRaw := range raw.Properties {
		var prop struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			continue
		}
		params.Properties[name] = ollama.ToolFunctionParamProperties{
			Type:        prop.Type,
			Description: prop.Description,
		}
	}
	params.Required = raw.Required

	return params
}

// ConnectToServer spawns the given command as an MCP subprocess, performs the MCP handshake,
// and discovers the server's tools. The caller owns the returned *ServerConn and must call
// Close() when done.
func ConnectToServer(ctx context.Context, name string, cmd *exec.Cmd) (*ServerConn, error) {
	client := mcp.NewClient(
		&mcp.Implementation{Name: "reagent", Version: "v0.1.0"},
		nil,
	)
	transport := &mcp.CommandTransport{Command: cmd}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", name, err)
	}

	conn := &ServerConn{Name: name, session: session}
	if err := conn.discoverTools(ctx); err != nil {
		_ = session.Close()
		return nil, err
	}

	return conn, nil
}
