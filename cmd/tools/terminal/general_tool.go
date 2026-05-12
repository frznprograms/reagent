// Package terminal implements the MCP server for terminal usage.
package terminal

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TerminalInput struct {
	Command         string `json:"command"         jsonschema:"the command to run, e.g. 'ls -la'"`
	AcceptLongInput bool   `json:"acceptLongInput" jsonschema:"if true, disables the 7000-character output cap"`
}

type TerminalOutput struct {
	Output string `json:"output" jsonschema:"stdout from the command, truncated to 7000 chars unless acceptLongInput is set"`
}

// buildCommandMap builds hash maps that operate as hash sets for the allowed command lists.
// This is purely for convenience of comparing commands at runtime.
func buildCommandMap() (map[string]bool, map[string]bool, map[string]bool) {
	var (
		allowedCommands   = [11]string{"ls", "cat", "echo", "pwd", "grep", "find", "wc", "head", "tail", "pytest", "make"}
		allowedGitSubcmds = [5]string{"status", "log", "diff", "show", "branch"}
		shellOperations   = [8]string{"&&", "||", "|", ";", "&", ">", "<", ">>"}
	)
	commandMap := make(map[string]bool)
	for _, cmd := range allowedCommands {
		commandMap[cmd] = true
	}
	gitMap := make(map[string]bool)
	for _, sub := range allowedGitSubcmds {
		gitMap[sub] = true
	}
	shellMap := make(map[string]bool)
	for _, op := range shellOperations {
		shellMap[op] = true
	}
	return commandMap, gitMap, shellMap
}

// verifyCommand checks the command the LLM wishes to run against the allowlist.
// LLMs are constrained to running at most one command per turn. While not as
// efficient, this helps with safety and constrains the LLM to reading files for
// additional context rather than chaining side-effecting commands.
func verifyCommand(command string) bool {
	tokens := strings.Fields(command)
	if len(tokens) < 1 {
		log.Println("LLM attempted to run an empty command. This is harmless, but unhelpful.")
		return true
	}
	commandMap, gitMap, shellMap := buildCommandMap()
	executable := strings.TrimSpace(tokens[0])

	if executable == "git" {
		if len(tokens) < 2 {
			return false
		}
		_, ok := gitMap[tokens[1]]
		return ok
	}

	if _, ok := commandMap[executable]; !ok {
		return false
	}

	// Reject shell operators and subshell expansions that would chain commands.
	for _, token := range tokens[1:] {
		if strings.HasPrefix(token, "$") || strings.HasPrefix(token, "`") {
			return false
		}
		if _, isShellOp := shellMap[strings.TrimSpace(token)]; isShellOp {
			return false
		}
	}

	return true
}

// RunTerminalCommand runs a command provided by the LLM in the terminal. It always
// checks the command against the allowlist first. Timeouts are handled by ctx.
// Output is truncated to 7000 chars to ensure the LLM receives meaningful context
// without being excessive; this can be overridden via input.AcceptLongInput.
func RunTerminalCommand(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input TerminalInput,
) (*mcp.CallToolResult, TerminalOutput, error) {
	if !verifyCommand(input.Command) {
		// Return as an error so the SDK sets IsError on the CallToolResult,
		// which signals the orchestrator to treat this as a tool failure rather
		// than crashing the entire server process via log.Fatalf.
		return nil, TerminalOutput{}, fmt.Errorf("command rejected: %q is not on the allowlist", input.Command)
	}

	log.Printf("Running command: %s", input.Command)

	tokens := strings.Fields(input.Command)
	cmd := exec.CommandContext(ctx, tokens[0], tokens[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil, TerminalOutput{}, fmt.Errorf("command failed: %w", err)
	}

	s := string(output)
	if !input.AcceptLongInput && len([]rune(s)) > 7000 {
		s = string([]rune(s)[:7000])
	}

	return nil, TerminalOutput{Output: s}, nil
}
