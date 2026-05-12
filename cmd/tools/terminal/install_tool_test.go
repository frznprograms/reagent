package terminal

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// These are integration tests: they invoke real package managers on the host.
// Run with: go test -tags integration ./...

type installCase struct {
	name       string
	packages   []string
	manager    string
	subcommand string
	wantErr    bool
}

// safetyCheckThenInstall runs the full gate: safety check → install.
// It mirrors what the agent orchestrator does at runtime.
func safetyCheckThenInstall(t *testing.T, ctx context.Context, ic installCase) error {
	t.Helper()
	req := &mcp.CallToolRequest{}

	_, safetyOut, err := RunInstallSafetyChecks(ctx, req, InstallSafetyCheckInput{
		Packages: ic.packages,
		Manager:  ic.manager,
	})
	if err != nil {
		return err
	}

	_, _, err = RunInstallCommand(ctx, req, InstallInput{
		Packages:   ic.packages,
		Manager:    ic.manager,
		Subcommand: ic.subcommand,
		Token:      safetyOut.Token,
	})
	return err
}

var installCases = []installCase{
	// --- uv ---
	{
		name:       "uv add single package",
		packages:   []string{"requests"},
		manager:    "uv",
		subcommand: "add",
	},
	{
		name:       "uv add multiple packages",
		packages:   []string{"pydantic", "httpx"},
		manager:    "uv",
		subcommand: "add",
	},
	{
		name:       "uv remove single package",
		packages:   []string{"requests"},
		manager:    "uv",
		subcommand: "remove",
	},

	// --- pip ---
	{
		name:       "pip install single package",
		packages:   []string{"rich"},
		manager:    "pip",
		subcommand: "install",
	},
	{
		name:       "pip install multiple packages",
		packages:   []string{"pydantic", "scikit-learn"},
		manager:    "pip",
		subcommand: "install",
	},

	// --- conda ---
	{
		name:       "conda install single package",
		packages:   []string{"numpy"},
		manager:    "conda",
		subcommand: "install",
	},
	{
		name:       "conda install multiple packages",
		packages:   []string{"pydantic", "pandas", "scikit-learn"},
		manager:    "conda",
		subcommand: "install",
	},

	// --- go ---
	{
		name:       "go get single module",
		packages:   []string{"github.com/joho/godotenv"},
		manager:    "go",
		subcommand: "get",
	},

	// --- gate enforcement ---
	{
		name:       "disallowed manager rejected",
		packages:   []string{"curl"},
		manager:    "apt",
		subcommand: "install",
		wantErr:    true,
	},
	{
		name:       "disallowed subcommand rejected",
		packages:   []string{"requests"},
		manager:    "pip",
		subcommand: "download", // not on the allowlist
		wantErr:    true,
	},
	{
		name:       "empty packages rejected",
		packages:   []string{},
		manager:    "uv",
		subcommand: "add",
		wantErr:    true,
	},
}

func TestRunInstallTool(t *testing.T) {
	ctx := context.Background()

	for _, ic := range installCases {
		t.Run(ic.name, func(t *testing.T) {
			err := safetyCheckThenInstall(t, ctx, ic)
			if ic.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTokenExpiry(t *testing.T) {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	_, safetyOut, err := RunInstallSafetyChecks(ctx, req, InstallSafetyCheckInput{
		Packages: []string{"requests"},
		Manager:  "pip",
	})
	if err != nil {
		t.Fatalf("safety check failed: %v", err)
	}

	// Manually expire the token.
	tokenMu.Lock()
	entry := tokenStore[safetyOut.Token]
	entry.expiresAt = time.Now().Add(-1 * time.Second)
	tokenStore[safetyOut.Token] = entry
	tokenMu.Unlock()

	_, _, err = RunInstallCommand(ctx, req, InstallInput{
		Packages:   []string{"requests"},
		Manager:    "pip",
		Subcommand: "install",
		Token:      safetyOut.Token,
	})
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestTokenSingleUse(t *testing.T) {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	_, safetyOut, err := RunInstallSafetyChecks(ctx, req, InstallSafetyCheckInput{
		Packages: []string{"rich"},
		Manager:  "pip",
	})
	if err != nil {
		t.Fatalf("safety check failed: %v", err)
	}

	input := InstallInput{
		Packages:   []string{"rich"},
		Manager:    "pip",
		Subcommand: "install",
		Token:      safetyOut.Token,
	}

	// First use should succeed.
	if _, _, err = RunInstallCommand(ctx, req, input); err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	// Second use with the same token must fail.
	if _, _, err = RunInstallCommand(ctx, req, input); err == nil {
		t.Error("expected error on second use of token, got nil")
	}
}

func TestTokenManagerMismatch(t *testing.T) {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	_, safetyOut, err := RunInstallSafetyChecks(ctx, req, InstallSafetyCheckInput{
		Packages: []string{"requests"},
		Manager:  "pip",
	})
	if err != nil {
		t.Fatalf("safety check failed: %v", err)
	}

	// Token was issued for pip but install attempts to use uv.
	_, _, err = RunInstallCommand(ctx, req, InstallInput{
		Packages:   []string{"requests"},
		Manager:    "uv",
		Subcommand: "add",
		Token:      safetyOut.Token,
	})
	if err == nil {
		t.Error("expected error for manager mismatch, got nil")
	}
}
