package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Data Structures ---

// ListAvailableServersInput is intentionally empty. This teaches the SDK to generate
// a JSON Schema that requires zero arguments from the LLM.
type ListAvailableServersInput struct{}

type ListAvailableServersOutput struct {
	Servers []string `json:"servers"`
}

// --- Tool Handler ---

func HandleListAvailableServers(ctx context.Context, req *mcp.CallToolRequest, input ListAvailableServersInput) (*mcp.CallToolResult, ListAvailableServersOutput, error) {
	app.LogIncomingRequest(req)

	result := make([]string, 0, len(databaseServersConfig))

	for serverName := range databaseServersConfig {
		result = append(result, serverName)
	}

	output := ListAvailableServersOutput{Servers: result}

	app.LogOutgoingResponse(output)

	return nil, output, nil
}
