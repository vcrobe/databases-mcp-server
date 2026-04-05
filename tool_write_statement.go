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
	SQL         string `json:"sql" jsonschema:"The raw SQL write statement (INSERT, UPDATE, DELETE, CREATE, DROP) to execute. CANNOT be a SELECT query. Qualify table references for the target engine (MySQL: database_name.table_name, PostgreSQL: schema_name.table_name)."`
}

type WriteStatementOutput struct {
	RowsAffected int64  `json:"rows_affected"`
	LastInsertID *int64 `json:"last_insert_id,omitempty"`
	Message      string `json:"message"`
}

// --- Tool Handler ---

func HandleExecuteWriteStatement(ctx context.Context, req *mcp.CallToolRequest, input WriteStatementInput) (*mcp.CallToolResult, WriteStatementOutput, error) {
	app.LogIncomingRequest(req)

	if input.ServerName == "" {
		return newErrorResult(errors.New("server_name is required")), WriteStatementOutput{Message: "server_name is required"}, nil
	}

	if strings.TrimSpace(input.Explanation) == "" {
		err := errors.New("explanation field is required and cannot be empty")
		logger.Printf("Validation failed: %v\n", err)
		return newErrorResult(err), WriteStatementOutput{Message: err.Error()}, nil
	}

	if strings.TrimSpace(input.SQL) == "" {
		err := errors.New("sql field is required and cannot be empty")
		logger.Printf("Validation failed: %v\n", err)
		return newErrorResult(err), WriteStatementOutput{Message: err.Error()}, nil
	}

	// Basic validation to prevent dangerous operations
	statementSplitted := strings.Fields(input.SQL)
	if len(statementSplitted) == 0 {
		err := errors.New("sql field is required and cannot be empty")
		logger.Printf("Validation failed: %v\n", err)
		return newErrorResult(err), WriteStatementOutput{Message: err.Error()}, nil
	}

	if strings.ToUpper(statementSplitted[0]) == "DROP" {
		if len(statementSplitted) < 2 {
			msg := "incomplete DROP statement is not allowed\n"
			logger.Print(msg)
			return newErrorResult(errors.New(strings.TrimSpace(msg))), WriteStatementOutput{Message: msg}, nil
		}

		tokenUpper := strings.ToUpper(statementSplitted[1])

		if tokenUpper == "TABLE" || tokenUpper == "DATABASE" {
			msg := fmt.Sprintf("DROP %s statement is not allowed\n", tokenUpper)
			logger.Print(msg)
			return newErrorResult(errors.New(strings.TrimSpace(msg))), WriteStatementOutput{Message: msg}, nil
		}
	}

	config, err := app.getServerConfig(input.ServerName)
	if err != nil {
		return newErrorResult(err), WriteStatementOutput{Message: err.Error()}, nil
	}

	db, err := app.getConn(input.ServerName)
	if err != nil {
		return newErrorResult(err), WriteStatementOutput{Message: err.Error()}, nil
	}

	// Execute the raw SQL string directly.
	result, err := db.ExecContext(ctx, input.SQL)
	if err != nil {
		logger.Printf("Write statement execution failed: %v\n", err)
		return newErrorResult(err), WriteStatementOutput{Message: err.Error()}, nil
	}

	rowsAffected, _ := result.RowsAffected()
	var lastInsertID *int64
	if id, err := result.LastInsertId(); err == nil {
		lastInsertID = &id
	}

	message := fmt.Sprintf("State modified successfully on %s server.", config.Engine)

	output := WriteStatementOutput{
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
		Message:      message,
	}

	app.LogOutgoingResponse(output)

	return nil, output, nil
}
