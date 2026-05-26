package terminal

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This file performs terminal operations related to installations. Since installations
// can be dangerous in rare cases, the agent must run RunInstallSafetyChecks first, which
// returns a structured query payload. The orchestrator routes that payload to the WebSearch
// tool. Only after reviewing results should the agent call RunInstallCommand.

// InstallSafetyCheckInput is the input for RunInstallSafetyChecks.
// Token is intentionally absent: it is issued by this tool, not provided to it.
// For more information on the token, see token.go
type InstallSafetyCheckInput struct {
	Packages []string `json:"packages" jsonschema:"packages to check, e.g. ['requests','numpy']"`
	Manager  string   `json:"manager"  jsonschema:"package manager to use, e.g. 'uv', 'pip', 'go'"`
}

// InstallInput is the input for RunInstallCommand.
// The Token field must be the value returned by a prior RunInstallSafetyChecks call.
type InstallInput struct {
	Packages   []string `json:"packages"   jsonschema:"packages to install"`
	Manager    string   `json:"manager"    jsonschema:"package manager to use"`
	Subcommand string   `json:"subcommand" jsonschema:"subcommand to run, e.g. 'add', 'install'"`
	Token      string   `json:"token"      jsonschema:"safety token returned by RunInstallSafetyChecks"`
}

// SafetyCheckOutput is returned by RunInstallSafetyChecks.
// The fields mirror SearchInputSchema in web_search/server.py so the orchestrator can pass
// them directly to the WebSearch tool without any transformation.
// Token must be threaded through to the subsequent RunInstallCommand call.
type SafetyCheckOutput struct {
	// WebSearch fields map 1:1 to SearchInputSchema in server.py.
	Query          string   `json:"query"           jsonschema:"security-focused search query for Tavily"`
	IncludeDomains []string `json:"include_domains" jsonschema:"domains to restrict results to"`
	SearchDepth    string   `json:"search_depth"    jsonschema:"'basic' or 'advanced'"`
	Topic          string   `json:"topic"           jsonschema:"'general', 'news', or 'finance'"`
	// Gate token must be forwarded to RunInstallCommand unchanged.
	Token string `json:"token" jsonschema:"single-use safety token; forward to RunInstallCommand"`
}

// InstallOutput is returned by RunInstallCommand.
type InstallOutput struct {
	Output string `json:"output" jsonschema:"combined stdout and stderr from the install command"`
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

// RunInstallSafetyChecks returns a structured Tavily search payload for the
// requested packages, plus a short-lived token that gates RunInstallCommand.
// The orchestrator must pass the query fields to the WebSearch tool and review
// results before proceeding with installation.
func RunInstallSafetyChecks(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input InstallSafetyCheckInput,
) (*mcp.CallToolResult, SafetyCheckOutput, error) {
	if len(input.Packages) == 0 {
		return nil, SafetyCheckOutput{}, fmt.Errorf("no packages specified")
	}
	installMap := buildInstallMap()
	if _, ok := installMap[input.Manager]; !ok {
		return nil, SafetyCheckOutput{}, fmt.Errorf("package manager %q is not permitted", input.Manager)
	}

	joined := strings.Join(input.Packages, " ")
	query := fmt.Sprintf("security vulnerabilities supply chain attack CVE %s", joined)

	token, err := generateToken()
	if err != nil {
		return nil, SafetyCheckOutput{}, fmt.Errorf("failed to generate safety token: %w", err)
	}

	tokenMu.Lock()
	tokenStore[token] = tokenEntry{
		packages:  input.Packages,
		manager:   input.Manager,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	tokenMu.Unlock()

	log.Printf("Install safety check requested for: %s", joined)

	return nil, SafetyCheckOutput{
		Query:          query,
		IncludeDomains: []string{"nvd.nist.gov", "snyk.io", "osv.dev", "github.com/advisories"},
		SearchDepth:    "advanced",
		Topic:          "general",
		Token:          token,
	}, nil
}

// RunInstallCommand runs an installation command. The agent must call
// RunInstallSafetyChecks, pass the returned query to WebSearch, and review the
// results before invoking this. The Token field from SafetyCheckOutput must be
// forwarded here unchanged; it is single-use and expires after 5 minutes.
func RunInstallCommand(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input InstallInput,
) (*mcp.CallToolResult, InstallOutput, error) {
	tokenMu.Lock()
	entry, ok := tokenStore[input.Token]
	if !ok {
		return nil, InstallOutput{}, fmt.Errorf("missing or invalid safety check token — run RunInstallSafetyChecks first")
	}

	delete(tokenStore, input.Token) // single-use: consumed on first call
	tokenMu.Unlock()

	if time.Now().After(entry.expiresAt) {
		return nil, InstallOutput{}, fmt.Errorf("safety check token expired — run RunInstallSafetyChecks again")
	}
	if entry.manager != input.Manager {
		return nil, InstallOutput{}, fmt.Errorf("token was issued for manager %q, not %q", entry.manager, input.Manager)
	}

	installMap := buildInstallMap()
	subcmds, managerOk := installMap[input.Manager]
	if !managerOk {
		return nil, InstallOutput{}, fmt.Errorf("package manager %q is not permitted", input.Manager)
	}
	if !subcmds[input.Subcommand] {
		return nil, InstallOutput{}, fmt.Errorf("subcommand %q is not permitted for %s", input.Subcommand, input.Manager)
	}

	args := append([]string{input.Subcommand}, input.Packages...)
	cmd := exec.CommandContext(ctx, input.Manager, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, InstallOutput{}, fmt.Errorf("install failed: %w\noutput: %s", err, string(output))
	}

	return nil, InstallOutput{Output: string(output)}, nil
}
