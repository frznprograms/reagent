// Package terminal is the entrypoint for the terminal MCP server.
package terminal

import (
	"context"
	"fmt"
	"log"

	"reagent/cmd/tools/terminal"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "terminal", Version: "v0.1.0"},
		&mcp.ServerOptions{
			Instructions: `Terminal tool server. Provides safe, restricted command execution and package
			installation with mandatory safety checks`,
			InitializedHandler: func(context.Context, *mcp.InitializedRequest) {
				fmt.Println("Initialised terminal tool.")
			},
			// consider eventually using ProgressNotificationHandler
		},
	)

	mcp.AddTool(server, &mcp.Tool{Name: "install", Description: "run an installation in the terminal"}, terminal.RunInstallCommand)
	mcp.AddTool(server, &mcp.Tool{Name: "install-safety-check", Description: "check safety of proposed installations"}, terminal.RunInstallSafetyChecks)
	mcp.AddTool(server, &mcp.Tool{Name: "run-command", Description: "run an approved terminal command (not related to installations)"}, terminal.RunTerminalCommand)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
