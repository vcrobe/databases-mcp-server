package main

import (
	"context"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Data Structures ---

type SelectQueryInput struct {
	ServerName  string `json:"server_name" jsonschema:"The name of the database server to execute the query against."`
	Explanation string `json:"explanation" jsonschema:"A brief, one-sentence explanation of why you are running this specific query, and what user question it answers. Failure to provide a clear reason will result in rejection."`
	SQL         string `json:"sql" jsonschema:"The raw SQL SELECT statement to execute. MUST strictly be a read-only query. Do NOT pass INSERT, UPDATE, DELETE, or CREATE statements. Qualify table references for the target engine (MySQL: database_name.table_name, PostgreSQL: schema_name.table_name)."`
}

type SelectQueryOutput struct {
	Rows []map[string]any `json:"rows"`
}

// --- Tool Handler ---

func HandleExecuteSelectQuery(ctx context.Context, req *mcp.CallToolRequest, input SelectQueryInput) (*mcp.CallToolResult, SelectQueryOutput, error) {
	app.LogIncomingRequest(req)

	if input.ServerName == "" {
		return newErrorResult(errors.New("server_name is required")), SelectQueryOutput{}, nil
	}

	if strings.TrimSpace(input.Explanation) == "" {
		err := errors.New("explanation field is required and cannot be empty")
		logger.Printf("Validation failed: %v\n", err)
		return newErrorResult(err), SelectQueryOutput{}, nil
	}

	db, err := app.getConn(input.ServerName)
	if err != nil {
		return newErrorResult(err), SelectQueryOutput{}, nil
	}

	rows, err := db.QueryContext(ctx, input.SQL)
	if err != nil {
		logger.Printf("Select query failed: %v\n", err)
		return newErrorResult(err), SelectQueryOutput{}, nil
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]any

	for rows.Next() {
		columns := make([]any, len(cols))
		columnPointers := make([]any, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			logger.Printf("Row scan failed: %v\n", err)
			return newErrorResult(err), SelectQueryOutput{}, nil
		}

		rowMap := make(map[string]any)
		for i, colName := range cols {
			val := columnPointers[i].(*any)
			rowMap[colName] = *val
		}
		results = append(results, rowMap)
	}

	output := SelectQueryOutput{Rows: results}
	app.LogOutgoingResponse(output)
	return nil, output, nil
}
