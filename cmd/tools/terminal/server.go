// Package terminal implements the MCP for terminal usage.
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
	Name            string `json:"name"`
	Command         string `json:"command"`
	AcceptLongInput bool   `json:"acceptLongInput"`
}

type TerminalOutput struct {
	Output string `json:"output"`
}

// buildCommandMap builds hash maps that operate more as hash sets for the allowed command lists.
// This is purely for convenience of comparing commands at runtime.
func buildCommandMap() (map[string]bool, map[string]bool, map[string]bool) {
	var (
		AllowedCommands   = [11]string{"ls", "cat", "echo", "pwd", "grep", "find", "wc", "head", "tail", "pytest", "make"}
		AllowedGitSubcmds = [5]string{"status", "log", "diff", "show", "branch"}
		ShellOperations   = [8]string{"&&", "||", "|", ";", "&", ">", "<", ">>"}
	)
	CommandMap := make(map[string]bool)
	for _, cmd := range AllowedCommands {
		CommandMap[cmd] = true
	}
	GitMap := make(map[string]bool)
	for _, sub := range AllowedGitSubcmds {
		GitMap[sub] = true
	}
	ShellMap := make(map[string]bool)
	for _, op := range ShellOperations {
		ShellMap[op] = true
	}

	return CommandMap, GitMap, ShellMap
}

// verifyCommand checks the command signature the LLM wishes to run. LLMs are constrained to
// running at most one command in single turn. While not as efficient, this helps with safety,
// and constrains the LLM to reading files for additional context.
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

	_, ok := commandMap[executable]
	if !ok {
		return false
	}

	// reject shell operators that would chain additional commands
	for _, token := range tokens[1:] {
		if strings.HasPrefix(token, "$") || strings.HasPrefix(token, "`") {
			return false
		}
		_, ok := shellMap[strings.TrimSpace(token)]
		if !ok {
			return true
		}
	}

	return false
}

// RunTerminalCommand runs a command provided by the LLM in the terminal. It always checks for
// the safety of the command first. Timeouts will be handled by ctx in main.
// Output is truncated to 7000 chars, to ensure the LLM receives meaningful context without
// being excessive. This can be overriden via input.AcceptLongInput.
func RunTerminalCommand(ctx context.Context, req *mcp.CallToolRequest, input TerminalInput) (
	*mcp.CallToolResult, TerminalOutput, error,
) {
	isCommandSafe := verifyCommand(input.Command)
	if !isCommandSafe {
		log.Fatalf("LLM attempted to run %s command, was rejected for being unsafe.", input.Command)
	}
	log.Printf("LLM intends to run %s ", input.Command)
	cmd := exec.Command(input.Name, input.Command)
	output, err := cmd.Output() // capture output
	if err != nil {
		fmt.Println("Unable to execute command: ", err)
	}
	s := string(output)
	runes := []rune(s)
	if len(runes) > 7000 && !input.AcceptLongInput {
		s = string(runes[:7000])
	}
	return nil, TerminalOutput{Output: s}, nil
}
