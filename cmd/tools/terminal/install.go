package terminal

// TODO: check how to use the prepared web search tool to integrate with this
// TODO: check if json schema from web_search.schemas needs to be defined here again

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This package performs terminal operations related to installations. Since installations
// can be dangerous in rare cases, the LLM can be asked to first check online for security
// updates before performing installations.

// buildInstallMap creates a map of acceptable tools to use for installations. The list
// of tools is narrow on purpose - agents should not be allowed to run installations through
// unverified channels.
func buildInstallMap() map[string]bool {
	AllowedCommands := []string{"uv", "pip", "conda", "go get"}
	CommandMap := make(map[string]bool)
	for _, cmd := range AllowedCommands {
		CommandMap[cmd] = true
	}
	return CommandMap
}

// RunInstallSafetyChecks is used in the installation tool, but calls the web
// search tool in turn, to ascertain from recent news whether the package(s)
// the user wishes to install are safe.
//
// It is left as its own tool, within the same server so it can be run independently.
func RunInstallSafetyChecks(ctx context.Context, req *mcp.CallToolRequest, input TerminalInput) (
	*mcp.CallToolResult, TerminalOutput, error,
) {
}

// RunInstallCommand runs a command provided by the LLM in the terminal, specifically to run
// installations. For safety, this is left as a separate concern in this codebase. Use of this
// tool automatically triggers a search into whether the desired packages for installation are
// safe to use.
func RunInstallCommand(ctx context.Context, req *mcp.CallToolRequest, input TerminalInput) (
	*mcp.CallToolResult, TerminalOutput, error,
) {
}
