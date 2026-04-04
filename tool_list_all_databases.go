package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Data Structures ---

// ListAllDatabasesInput is intentionally empty. This teaches the SDK to generate
// a JSON Schema that requires zero arguments from the LLM.
type ListAllDatabasesInput struct {
	ServerName string `json:"server_name" jsonschema:"The name of the database server to connect to, as defined in the environment variables."`
}

type ListAllDatabasesOutput struct {
	Databases []string `json:"databases"`
}

// --- Tool Handler ---

func HandleListAllDatabases(ctx context.Context, req *mcp.CallToolRequest, input ListAllDatabasesInput) (*mcp.CallToolResult, ListAllDatabasesOutput, error) {
	app.LogIncomingRequest(req)

	if input.ServerName == "" {
		return newErrorResult(fmt.Errorf("server_name is required")), ListAllDatabasesOutput{}, nil
	}

	db, err := app.getConn(input.ServerName)
	if err != nil {
		return newErrorResult(err), ListAllDatabasesOutput{}, nil
	}

	rows, err := db.QueryContext(ctx, "SHOW DATABASES")

	if err != nil {
		logger.Printf("List databases query failed: %v\n", err)
		return newErrorResult(err), ListAllDatabasesOutput{}, nil
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var databaseName string
		if err := rows.Scan(&databaseName); err != nil {
			logger.Printf("Row scan failed: %v\n", err)
			return newErrorResult(err), ListAllDatabasesOutput{}, nil
		}
		databases = append(databases, databaseName)
	}

	output := ListAllDatabasesOutput{Databases: databases}

	app.LogOutgoingResponse(output)

	return nil, output, nil
}
