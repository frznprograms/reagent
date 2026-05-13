// Command agent is the entry point for the reagent agent.
// It connects to the terminal and web-search MCP servers, then runs a
// single query supplied on the command line:
//
//	go run ./cmd/agent "what Go version introduced range-over-integers?"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"reagent/cmd/agent"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: agent <query>")
		os.Exit(1)
	}
	query := strings.Join(os.Args[1:], " ")

	ctx := context.Background()

	// Start MCP servers
	// Terminal server: the Go binary built from the terminal package.
	terminalConn, err := agent.ConnectToServer(ctx, "terminal",
		exec.Command("./bin/terminal-server"),
	)
	if err != nil {
		log.Fatalf("terminal server: %v", err)
	}
	defer terminalConn.Close()

	// Web-search server: the FastMCP Python server.
	webConn, err := agent.ConnectToServer(ctx, "websearch",
		exec.Command("python", "-m", "reagent.tools.websearch"),
	)
	if err != nil {
		log.Fatalf("websearch server: %v", err)
	}
	defer webConn.Close()

	// create LM
	// Reads OLLAMA_HOST from the environment if set; falls back to localhost.
	ollamaURL := os.Getenv("OLLAMA_HOST")
	llm, err := agent.NewLLM(ollamaURL, "") // "" → DefaultModel (qwen3:8b)
	if err != nil {
		log.Fatalf("llm: %v", err)
	}

	// run the agent
	a := agent.New(llm, []*agent.ServerConn{terminalConn, webConn})

	answer, err := a.Run(ctx, query)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}

	fmt.Println(answer)
}
