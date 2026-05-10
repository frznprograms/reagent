package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This package performs terminal operations related to installations. Since installations
// can be dangerous in rare cases, the agent must run RunInstallSafetyChecks first, which
// returns a web search query. The orchestrator routes that query to the WebSearch tool.
// Only after reviewing results should the agent call RunInstallCommand.

type tokenEntry struct {
	packages  []string
	manager   string
	expiresAt time.Time
}

var (
	tokenMu    sync.Mutex
	tokenStore = make(map[string]tokenEntry)
)

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type InstallInput struct {
	Packages   []string `json:"packages"`
	Manager    string   `json:"manager"`
	Subcommand string   `json:"subcommand"`
	Token      string   `json:"token"`
}

type SafetyCheckOutput struct {
	WebSearchQuery string `json:"webSearchQuery"`
	Token          string `json:"token"`
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

	// FIXME: this prompt needs to be adjusted to fit tavily schema
	query := fmt.Sprintf(
		`security vulnerabilities supply chain attack CVE %s 
		site:nvd.nist.gov OR site:snyk.io OR site:osv.dev OR site:github.com/advisories`,
		joined,
	)

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
	return nil, SafetyCheckOutput{WebSearchQuery: query, Token: token}, nil
}

// FIXME: return value of all tools should be packaged in the *mcp.CallToolResult!!

// RunInstallCommand runs an installation command. The agent must call RunInstallSafetyChecks
// and review the web search results before invoking this.
func RunInstallCommand(ctx context.Context, req *mcp.CallToolRequest, input InstallInput) (
	*mcp.CallToolResult, TerminalOutput, error,
) {
	// verify token signature
	tokenMu.Lock()
	entry, ok := tokenStore[input.Token]
	delete(tokenStore, input.Token) // single-use: consumed on first call
	tokenMu.Unlock()
	if !ok {
		return nil, TerminalOutput{}, fmt.Errorf("missing or invalid safety check token - run RunInstallSafetyChecks")
	}
	if time.Now().After(entry.expiresAt) {
		return nil, TerminalOutput{}, fmt.Errorf("safety check token expired - run RunInstallSafetyChecks again")
	}
	if entry.manager != input.Manager {
		return nil, TerminalOutput{}, fmt.Errorf("token was issued for manager %q, not %q", entry.manager, input.Manager)
	}

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
