// Package clients creates the mcp client, which connects to the mcp servers
// for tool use if required.
package clients

import (
	"context"
	"log"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// func connectToTavilyMCP(ctx context.Context) (*mcp.ClientSession, error) {
// }

func main() {
	ctx := context.Background()

	// connect a new client, with no features
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v0.1.0"}, nil)

	// connect to server over stdio
	transport := &mcp.CommandTransport{Command: exec.Command("myserver")}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// call a tool on the server
	// params := &mcp.CallToolParams{
	// 	Name:      "greet",
	// 	Arguments: map[string]any{"name": "you"},
	// }
	// res, err := session.CallTool(ctx, params)
	// if err != nil {
	// 	log.Fatalf("CallTool failed: %v", err)
	// }
	// if res.IsError {
	// 	log.Fatal("tool failed")
	// }
	// for _, c := range res.Content {
	// 	log.Print(c.(*mcp.TextContent).Text)
	// }
}
