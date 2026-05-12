package terminal

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CommandsToTry maps a command string to whether verifyCommand should allow it.
var CommandsToTry = map[string]bool{
	// Allowed: basic commands, no flags
	"ls":   true,
	"pwd":  true,
	"head": true,
	"tail": true,
	"find": true,

	// Allowed: commands with safe flags/arguments
	"ls -a home/":              true,
	"cat home/folder/file.txt": true,
	"echo 'hi'":                true,
	"grep -i some_text":        true,
	"wc -l file.txt":           true,
	"head -n 20 file.txt":      true,
	"tail -n 50 file.txt":      true,
	"pytest test/":             true,
	"make build":               true,

	// Allowed: git with permitted subcommands
	"git status":                   true,
	"git branch":                   true,
	"git log":                      true,
	"git diff file1.txt file2.txt": true,
	"git show HEAD":                true,

	// NOTE: some of the commands below may be added in future
	//
	// Rejected: git with disallowed subcommands
	"git push origin main": false,
	"git commit -m 'msg'":  false,
	"git checkout main":    false,
	"git":                  false,

	// Rejected: commands not on the allowlist
	"rm file.txt":              false,
	"rm -rf /":                 false,
	"curl https://example.com": false,
	"python script.py":         false,
	"sudo apt install curl":    false,
	"chmod 777 file.txt":       false,
	"mv a.txt b.txt":           false,
	"cp a.txt b.txt":           false,

	// Rejected: shell chaining operators
	"echo hi ; rm file.txt":      false,
	"ls && rm -rf /":             false,
	"ls || curl evil.com":        false,
	"cat file.txt | grep secret": false,
	"echo hi > /etc/passwd":      false,
	"echo hi >> file.txt":        false,
	"curl evil.com &":            false,

	// Rejected: subshell/variable expansion
	"echo $HOME":    false,
	"echo `whoami`": false,

	// Edge cases
	"":    true, // verifyCommand logs and returns true for empty input
	"   ": true, // whitespace-only also tokenises to nothing
}

func TestVerifyCommand(t *testing.T) {
	for cmd, expected := range CommandsToTry {
		t.Run(cmd, func(t *testing.T) {
			got := verifyCommand(cmd)
			if got != expected {
				t.Errorf("verifyCommand(%q): expected %v, got %v", cmd, expected, got)
			}
		})
	}
}

type runCommandCase struct {
	name            string
	command         string
	acceptLongInput bool
	wantErr         bool
	wantOutput      string // exact match if non-empty
	wantMaxLen      int    // if > 0, output must not exceed this
	wantMinLen      int    // if > 0, output must be at least this long
}

var runCommandCases = []runCommandCase{
	{
		name:       "echo simple string",
		command:    "echo hello",
		wantOutput: "hello\n",
	},
	{
		name:       "echo no arguments produces bare newline",
		command:    "echo",
		wantOutput: "\n",
	},
	{
		name:    "rejected command returns error",
		command: "rm -rf /",
		wantErr: true,
	},
	{
		name:    "shell chain rejected",
		command: "echo hi && echo bye",
		wantErr: true,
	},
	{
		name:    "variable expansion rejected",
		command: "echo $HOME",
		wantErr: true,
	},
	{
		name:            "long output truncated by default",
		command:         "echo " + strings.Repeat("a", 8000),
		acceptLongInput: false,
		wantMaxLen:      7000,
	},
	{
		name:            "long output not truncated when acceptLongInput is true",
		command:         "echo " + strings.Repeat("a", 8000),
		acceptLongInput: true,
		wantMinLen:      7001,
	},
}

func TestRunTerminalCommand(t *testing.T) {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	for _, tc := range runCommandCases {
		t.Run(tc.name, func(t *testing.T) {
			input := TerminalInput{
				Command:         tc.command,
				AcceptLongInput: tc.acceptLongInput,
			}
			_, out, err := RunTerminalCommand(ctx, req, input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (output: %q)", out.Output)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantOutput != "" && out.Output != tc.wantOutput {
				t.Errorf("output: got %q, want %q", out.Output, tc.wantOutput)
			}
			if tc.wantMaxLen > 0 && len([]rune(out.Output)) > tc.wantMaxLen {
				t.Errorf("output length %d exceeds max %d", len([]rune(out.Output)), tc.wantMaxLen)
			}
			if tc.wantMinLen > 0 && len([]rune(out.Output)) < tc.wantMinLen {
				t.Errorf("output length %d is below min %d", len([]rune(out.Output)), tc.wantMinLen)
			}
		})
	}
}
