package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// validTableName allows only safe identifiers to prevent SQL injection when
// the table name must be embedded directly in a DDL statement.
var validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// validDatabaseName applies the same safe-identifier rule to database names.
var validDatabaseName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// --- Data Structures ---

type InspectTableInput struct {
	ServerName   string `json:"server_name" jsonschema:"The name of the database server."`
	DatabaseName string `json:"database_name" jsonschema:"The name of the database that contains the table."`
	TableName    string `json:"table_name" jsonschema:"The exact name of the table to inspect. MUST be a single table name, not a SQL query."`
}

type InspectTableOutput struct {
	Schema string `json:"schema"`
}

// --- Tool Handler ---

func HandleInspectSingleTable(ctx context.Context, req *mcp.CallToolRequest, input InspectTableInput) (*mcp.CallToolResult, InspectTableOutput, error) {
	app.LogIncomingRequest(req)

	if input.ServerName == "" {
		return newErrorResult(fmt.Errorf("server_name is required")), InspectTableOutput{}, nil
	}

	if !validDatabaseName.MatchString(input.DatabaseName) {
		return newErrorResult(fmt.Errorf("invalid database name: %q", input.DatabaseName)), InspectTableOutput{}, nil
	}

	if !validTableName.MatchString(input.TableName) {
		return newErrorResult(fmt.Errorf("invalid table name: %q", input.TableName)), InspectTableOutput{}, nil
	}

	db, err := app.getConn(input.ServerName)
	if err != nil {
		return newErrorResult(err), InspectTableOutput{}, nil
	}

	var schema string
	// SHOW CREATE TABLE returns (Table, Create Table) — scan both columns.
	var tableName string
	err = db.QueryRowContext(ctx, "SHOW CREATE TABLE `"+input.DatabaseName+"`.`"+input.TableName+"`").Scan(&tableName, &schema)

	if err != nil {
		if err == sql.ErrNoRows {
			logger.Printf("Schema request failed: Table '%s.%s' does not exist.\n", input.DatabaseName, input.TableName)
			return newErrorResult(fmt.Errorf("table '%s.%s' does not exist in the database", input.DatabaseName, input.TableName)), InspectTableOutput{}, nil
		}
		logger.Printf("Database error fetching schema: %v\n", err)
		return newErrorResult(err), InspectTableOutput{}, nil
	}

	output := InspectTableOutput{Schema: schema}
	app.LogOutgoingResponse(output)
	return nil, output, nil
}
