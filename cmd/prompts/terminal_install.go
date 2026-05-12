// Package prompts contains reusable prompt templates for agent tool flows.
package prompts

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TODO: add prompts to server

func SafeInstallPrompt() *mcp.Prompt {
	prompt := &mcp.Prompt{
		Name:        "audit-package",
		Description: "Start a security audit conversation for a given package",
		Arguments: []*mcp.PromptArgument{
			{Name: "package", Description: "Package name to audit", Required: true},
			{Name: "package manager", Description: "Pacakge manager e.g. pip, uv, conda, npm"},
		},
	}

	return prompt
}

func SafeInstallPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	pkg := req.Params.Arguments
	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role:    mcp.Role("user"),
				Content: &mcp.TextContent{Text: fmt.Sprintf("I want to install the %s package. Run a safety check first, then install if safe.", pkg)},
			},
		},
	}, nil
}
