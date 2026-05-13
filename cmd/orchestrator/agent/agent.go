package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	ollama "github.com/ollama/ollama/api"
)

const (
	// maxTurns limits the agent loop to prevent runaway tool chains.
	// TODO: 20 may be too low
	maxTurns     = 20
	systemPrompt = `You are a reserch agent. You have access to tools for searching th web
	and running safe terminal commands. Use them to answer the user's questions as thoroughly
	and accurately as possible. When you have enough information to give a complete answer,
	stop calling tools and respond directly to the user.`
)

// Agent is the top-level orchestrator. It owns the LLM and the set of connected
// MCP servers.
type Agent struct {
	llm     *LLM
	servers []*ServerConn

	// allTools is the merged Ollama tool list from all connected servers.
	// built once at construction and reused across calls.
	allTools []ollama.Tool

	// toolIndex maps tool name -> the server that owns it for O(1) dispatch.
	toolIndex map[string]*ServerConn
}

// New constructs an Agent. Servers must already be connected (see ConnectToServer).
// The caller retains ownership of each ServerConn and is responsible for closing
// them when done.
func New(llm *LLM, servers []*ServerConn) *Agent {
	allTools := make([]ollama.Tool, 0)
	toolIndex := make(map[string]*ServerConn)

	for _, srv := range servers {
		for _, t := range srv.OllamaTools() {
			// Last writer wins on name collision; write warning if it happens
			if existing, dup := toolIndex[t.Function.Name]; dup {
				log.Printf("warning: tool %q already registerd by %s; %s will override it",
					t.Function.Name, existing.Name, srv.Name)
			}
			toolIndex[t.Function.Name] = srv
			allTools = append(allTools, t)
		}
	}

	return &Agent{
		llm:       llm,
		servers:   servers,
		allTools:  allTools,
		toolIndex: toolIndex,
	}
}

// Run processes a single user query and returns the agent's final text answer.
// Internally it runs the standard tool-calling loop:
// 1. Send messages + tools to the LM
// 2. If the model returns tool calls, dispatch each one via the owning MCP server and append results to history
// 3. Repeat until model replies with content and no tool calls, or until maxTurns is hit
func (a *Agent) Run(ctx context.Context, userQuery string) (string, error) {
	messages := []ollama.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userQuery},
	}
	for turn := range maxTurns {
		log.Printf("[agent] turn %d/%d", turn+1, maxTurns)
		reply, err := a.llm.Chat(ctx, messages, a.allTools)
		if err != nil {
			return "", fmt.Errorf("turn %d: %w", turn+1, err)
		}

		// always append assistant turn so history stays consistent
		messages = append(messages, reply)

		// no tool calls means model is done; return its text
		if len(reply.ToolCalls) == 0 {
			return reply.Content, nil
		}

		// dispatch every tool call and collect results
		for _, call := range reply.ToolCalls {
			result, dispatchErr := a.dispatch(ctx, call)

			// append the tool result regardless of error. If the call failed,
			// send the error text back so the model can decide what to do (e.g. retry,
			// give up, explain failure to user etc.)
			messages = append(messages, ollama.Message{
				Role:    "tool",
				Content: result,
			})
			if dispatchErr != nil {
				log.Printf("[agent] tool %q failed: %v", call.Function.Name, dispatchErr)
			}
		}
	}
	return "", fmt.Errorf("agent exceeded %d turns without a final answer", maxTurns)
}

// dispatch routes a single tool call to the correct MCP server and returns
// the text result (or an error string if the call itself failed).
func (a *Agent) dispatch(ctx context.Context, call ollama.ToolCall) (string, error) {
	name := call.Function.Name
	srv, ok := a.toolIndex[name]
	if !ok {
		msg := fmt.Sprintf("no server registered for tool %q", name)
		return msg, fmt.Errorf(msg)
	}

	// Ollama gives us arguments as a map[string]any already parsed from JSON.
	args, err := toStringAnyMap(call.Function.Arguments)
	if err != nil {
		msg := fmt.Sprintf("could not parse arguments for %q: %v", name, err)
		return msg, fmt.Errorf(msg)
	}

	log.Printf("[agent] calling tool %q on server %q with args %v", name, srv.Name, args)

	text, err := srv.Call(ctx, name, args)
	if err != nil {
		return err.Error(), err
	}
	return text, nil
}

// toStringAnyMap normalises the tool call arguments into map[string]any.
// Ollama's ToolCallFunction.Arguments is already typed as map[string]any,
// but the field is an any internally, so we round-trip through JSON to be safe.
func toStringAnyMap(v any) (map[string]any, error) {
	if m, ok := v.(map[string]any); ok {
		return m, nil
	}
	// Fallback: marshal then unmarshal.
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
