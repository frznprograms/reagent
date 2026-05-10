// Package terminal implements the MCP for terminal usage.
package terminal

import (
	"context"
	"go/build"
	"log"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TerminalInput struct {
	Name string `json:"name"`
	Command string `json:"command"`
}

type TerminalOutput struct {
	Output string `json:"output"`
}

const (
	AllowedCommands = [9]string{"ls", "cat", "echo", "pwd", "grep", "find", "wc", "head", "tail"}
	ShellOperations = [8]string{"&&", "||", "|", ";", "&", ">", "<", ">>"}
)

// buildCommandMap builds hash maps that operate more as hash sets for the AllowedCommands
// and ShellOperations consts. This is purely for convenience of comparing commands at runtime.
func buildCommandMap() (map[string]bool, map[string]bool){
	CommandMap := make(map[string]bool)
	for _, cmd := range AllowedCommands {
		CommandMap[cmd] = true
	}
	ShellMap := make(map[string]bool)
	for _, cmd := ShellOperations {
		ShellMap[cmd] = true
	}

	return CommandMap, ShellMap
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
	commandMap, shellMap := buildCommandMap()
	executable := strings.TrimSpace(tokens[0])
	// reject commands other than those which are pre-approved
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
// being excessive.
func RunTerminalCommand(ctx context.Context, req *mcp.CallToolRequest, input TerminalInput) (
	*mcp.CallToolResult, TerminalOutput, error,
) {
	isCommandSafe := verifyCommand(input.Command)
	if !isCommandSafe {
		log.Fatal("LLM attempted to run %s command, was rejected for being unsafe.", input.Command)
	}
	log.Printf("LLM intends to run %s ", input.Command)
	cmd := exec.Command(input.Name, input.Command)
	output, err := cmd.Output() // capture output
	if err != nil {
		fmt.Println("Unable to execute command: ", err)
	}
	return nil, TerminalOutput{Output: string(output[:7000])}, nil
}
