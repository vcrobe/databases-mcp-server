package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Data Structures ---

type WriteStatementInput struct {
	ServerName  string `json:"server_name" jsonschema:"The name of the database server to execute the statement against."`
	Explanation string `json:"explanation" jsonschema:"A brief, one-sentence explanation of why you are running this specific write statement, and what user question it answers. Failure to provide a clear reason will result in rejection."`
	SQL         string `json:"sql" jsonschema:"The raw SQL write statement (INSERT, UPDATE, DELETE, CREATE, DROP) to execute. CANNOT be a SELECT query. Because no default database is selected, you MUST qualify all table references as database_name.table_name (e.g. INSERT INTO mydb.users ...)."`
}

type WriteStatementOutput struct {
	RowsAffected int64  `json:"rows_affected"`
	LastInsertID int64  `json:"last_insert_id"`
	Message      string `json:"message"`
}

// --- Tool Handler ---

func HandleExecuteWriteStatement(ctx context.Context, req *mcp.CallToolRequest, input WriteStatementInput) (*mcp.CallToolResult, WriteStatementOutput, error) {
	app.LogIncomingRequest(req)

	if input.ServerName == "" {
		return newErrorResult(errors.New("server_name is required")), WriteStatementOutput{}, nil
	}

	if strings.TrimSpace(input.Explanation) == "" {
		err := errors.New("explanation field is required and cannot be empty")
		logger.Printf("Validation failed: %v\n", err)
		return newErrorResult(err), WriteStatementOutput{}, nil
	}

	// Basic validation to prevent dangerous operations
	statementSplitted := strings.Fields(input.SQL)
	if strings.ToUpper(statementSplitted[0]) == "DROP" {
		tokenUpper := strings.ToUpper(statementSplitted[1])

		if tokenUpper == "TABLE" || tokenUpper == "DATABASE" {
			msg := fmt.Sprintf("DROP %s statement is not allowed\n", tokenUpper)
			logger.Print(msg)
			return newErrorResult(errors.New(msg)), WriteStatementOutput{}, nil
		}
	}

	db, err := app.getConn(input.ServerName)
	if err != nil {
		return newErrorResult(err), WriteStatementOutput{}, nil
	}

	// Execute the raw SQL string directly.
	result, err := db.ExecContext(ctx, input.SQL)
	if err != nil {
		logger.Printf("Write statement execution failed: %v\n", err)
		return newErrorResult(err), WriteStatementOutput{}, nil
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	output := WriteStatementOutput{
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
		Message:      "State modified successfully.",
	}

	app.LogOutgoingResponse(output)

	return nil, output, nil
}
