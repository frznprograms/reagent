package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"reagent/cmd/tools/websearch"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func RunTavilyClient(ctx context.Context, req websearch.SearchRequest) (*websearch.SearchResponse, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "tavily-client", Version: "v0.1.0"}, &mcp.ClientOptions{})
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Tavily API Key from environment")
	}

	tavilyAPIKey := getEnv("TAVILY_API_KEY", "")
	if tavilyAPIKey == "" {
		return nil, fmt.Errorf("a valid Tavily API key must be provided, got %s instead", tavilyAPIKey)
	}

	url := fmt.Sprintf("https://mcp.tavily.com/mcp/?tavilyAPIKey=%s", tavilyAPIKey)
	transport := &mcp.SSEClientTransport{Endpoint: url}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Tavily server: %w", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("failed to close Tavily session gracefully: %v", err)
			log.Fatal()
		}
	}()

	result, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "tavily-search",
		Arguments: req,
	})

	var searchResp websearch.SearchResponse
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search response: %w", err)
	}

	return &searchResp, nil
}
