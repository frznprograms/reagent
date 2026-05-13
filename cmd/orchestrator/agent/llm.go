package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	ollama "github.com/ollama/ollama/api"
)

const (
	DefaultModel   = "qwen3:8b"
	DefaultBaseURL = "http://localhost:11434"
)

// LLM wraps the Ollama API client with the small set of behaviour the agent
// loop actually needs: a streaming-off chat call that returns the full
// assistant message in one shot. Streaming can be added later without
// changing the loop.
type LLM struct {
	client *ollama.Client
	model  string
}

// NewLLM constructs an LLM pointed at a local Ollama instance.
// Pass an empty string for baseURL or model to use the defaults.
func NewLLM(baseURL, model string) (*LLM, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if model == "" {
		model = DefaultModel
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama base URL %q: %w", baseURL, err)
	}

	client := ollama.NewClient(u, http.DefaultClient)
	return &LLM{client: client, model: model}, nil
}

// Chat sends the full conversation history to Ollama and blocks until the
// model finishes generating. tools may be nil or empty for a plain chat.
//
// The returned ollama.Message contains:
//   - .Content   — the assistant's text reply (may be empty if it only called tools)
//   - .ToolCalls — the list of tool invocations the model wants to make
func (l *LLM) Chat(
	ctx context.Context,
	messages []ollama.Message,
	tools []ollama.Tool,
) (ollama.Message, error) {
	req := &ollama.ChatRequest{
		Model:    l.model,
		Messages: messages,
		Tools:    tools,
		// Disable streaming: the callback is called exactly once with the
		// complete message, keeping the agent loop straightforward.
		Stream: boolPtr(false),
	}

	var reply ollama.Message
	err := l.client.Chat(ctx, req, func(resp ollama.ChatResponse) error {
		reply = resp.Message
		return nil
	})
	if err != nil {
		return ollama.Message{}, fmt.Errorf("ollama chat: %w", err)
	}

	return reply, nil
}

func boolPtr(b bool) *bool { return &b }
