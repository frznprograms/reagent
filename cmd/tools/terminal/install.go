package terminal

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This package performs terminal operations related to installations. Since installations
// can be dangerous in rare cases, the agent must run RunInstallSafetyChecks first, which
// returns a web search query. The orchestrator routes that query to the WebSearch tool.
// Only after reviewing results should the agent call RunInstallCommand.

type InstallInput struct {
	Packages   []string `json:"packages"`
	Manager    string   `json:"manager"`
	Subcommand string   `json:"subcommand"`
}

type SafetyCheckOutput struct {
	WebSearchQuery string `json:"webSearchQuery"`
}

// buildInstallMap returns a map of manager → allowed subcommands. Keeping the list
// narrow ensures agents can only use verified package managers and safe subcommands.
func buildInstallMap() map[string]map[string]bool {
	allowed := map[string][]string{
		"uv":    {"add", "init", "remove", "sync", "run"},
		"pip":   {"install", "uninstall", "list", "show"},
		"conda": {"install", "init", "create", "list"},
		"go":    {"get"},
	}
	result := make(map[string]map[string]bool, len(allowed))
	for manager, subcmds := range allowed {
		result[manager] = make(map[string]bool, len(subcmds))
		for _, s := range subcmds {
			result[manager][s] = true
		}
	}
	return result
}

// RunInstallSafetyChecks returns a targeted web search query for the requested packages.
// The orchestrator should route this query to the WebSearch tool and present results to
// the agent before allowing RunInstallCommand to proceed.
func RunInstallSafetyChecks(ctx context.Context, req *mcp.CallToolRequest, input InstallInput) (
	*mcp.CallToolResult, SafetyCheckOutput, error,
) {
	if len(input.Packages) == 0 {
		return nil, SafetyCheckOutput{}, fmt.Errorf("no packages specified")
	}
	installMap := buildInstallMap()
	if _, ok := installMap[input.Manager]; !ok {
		return nil, SafetyCheckOutput{}, fmt.Errorf("package manager %q is not permitted", input.Manager)
	}
	joined := strings.Join(input.Packages, ", ")
	query := fmt.Sprintf(
		`security vulnerabilities supply chain attack CVE %s 
		site:nvd.nist.gov OR site:snyk.io OR site:osv.dev OR site:github.com/advisories`,
		joined,
	)
	log.Printf("Install safety check requested for: %s", joined)
	return nil, SafetyCheckOutput{WebSearchQuery: query}, nil
}

// RunInstallCommand runs an installation command. The agent must call RunInstallSafetyChecks
// and review the web search results before invoking this.
func RunInstallCommand(ctx context.Context, req *mcp.CallToolRequest, input InstallInput) (
	*mcp.CallToolResult, TerminalOutput, error,
) {
	installMap := buildInstallMap()
	subcmds, managerOk := installMap[input.Manager]
	if !managerOk {
		return nil, TerminalOutput{}, fmt.Errorf("package manager %q is not permitted", input.Manager)
	}
	if !subcmds[input.Subcommand] {
		return nil, TerminalOutput{}, fmt.Errorf("subcommand %q is not permitted for %s", input.Subcommand, input.Manager)
	}

	args := append([]string{input.Subcommand}, input.Packages...)
	cmd := exec.CommandContext(ctx, input.Manager, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, TerminalOutput{Output: string(output)}, fmt.Errorf("install failed: %w", err)
	}
	return nil, TerminalOutput{Output: string(output)}, nil
}
