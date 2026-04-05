package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

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
	DatabaseName string `json:"database_name" jsonschema:"The namespace containing the table (MySQL database name or PostgreSQL schema name)."`
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

	config, err := app.getServerConfig(input.ServerName)
	if err != nil {
		return newErrorResult(err), InspectTableOutput{}, nil
	}

	db, err := app.getConn(input.ServerName)
	if err != nil {
		return newErrorResult(err), InspectTableOutput{}, nil
	}

	var schema string
	if config.Engine == "mysql" {
		schema, err = inspectMySQLTableSchema(ctx, db, input.DatabaseName, input.TableName)
	} else {
		schema, err = inspectPostgresTableSchema(ctx, db, input.DatabaseName, input.TableName)
	}

	if err != nil {
		return newErrorResult(err), InspectTableOutput{}, nil
	}

	output := InspectTableOutput{Schema: schema}
	app.LogOutgoingResponse(output)
	return nil, output, nil
}

func inspectMySQLTableSchema(ctx context.Context, db *sql.DB, databaseName string, tableName string) (string, error) {
	var schema string
	// SHOW CREATE TABLE returns (Table, Create Table) — scan both columns.
	var returnedTableName string
	err := db.QueryRowContext(ctx, "SHOW CREATE TABLE `"+databaseName+"`.`"+tableName+"`").Scan(&returnedTableName, &schema)

	if err != nil {
		if err == sql.ErrNoRows {
			logger.Printf("Schema request failed: Table '%s.%s' does not exist.\n", databaseName, tableName)
			return "", fmt.Errorf("table '%s.%s' does not exist in the database", databaseName, tableName)
		}
		logger.Printf("Database error fetching schema: %v\n", err)
		return "", err
	}

	return schema, nil
}

func inspectPostgresTableSchema(ctx context.Context, db *sql.DB, schemaName string, tableName string) (string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, schemaName, tableName)
	if err != nil {
		logger.Printf("PostgreSQL schema query failed: %v\n", err)
		return "", err
	}
	defer rows.Close()

	type columnMeta struct {
		Name       string
		DataType   string
		IsNullable string
		DefaultVal sql.NullString
	}

	columns := make([]columnMeta, 0, 8)
	for rows.Next() {
		var c columnMeta
		if err := rows.Scan(&c.Name, &c.DataType, &c.IsNullable, &c.DefaultVal); err != nil {
			logger.Printf("PostgreSQL schema row scan failed: %v\n", err)
			return "", err
		}
		columns = append(columns, c)
	}

	if err := rows.Err(); err != nil {
		logger.Printf("PostgreSQL schema iteration failed: %v\n", err)
		return "", err
	}

	if len(columns) == 0 {
		return "", fmt.Errorf("table '%s.%s' does not exist in the database", schemaName, tableName)
	}

	pkRows, err := db.QueryContext(ctx, `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(i.indkey)
		WHERE i.indisprimary AND n.nspname = $1 AND t.relname = $2
		ORDER BY a.attnum
	`, schemaName, tableName)
	if err != nil {
		logger.Printf("PostgreSQL primary key query failed: %v\n", err)
		return "", err
	}
	defer pkRows.Close()

	primaryKeys := make([]string, 0, 4)
	for pkRows.Next() {
		var keyName string
		if err := pkRows.Scan(&keyName); err != nil {
			logger.Printf("PostgreSQL primary key row scan failed: %v\n", err)
			return "", err
		}
		primaryKeys = append(primaryKeys, quoteIdentifierPostgres(keyName))
	}

	if err := pkRows.Err(); err != nil {
		logger.Printf("PostgreSQL primary key iteration failed: %v\n", err)
		return "", err
	}

	var b strings.Builder
	b.WriteString("CREATE TABLE ")
	b.WriteString(quoteIdentifierPostgres(schemaName))
	b.WriteString(".")
	b.WriteString(quoteIdentifierPostgres(tableName))
	b.WriteString(" (\n")

	for i, col := range columns {
		b.WriteString("  ")
		b.WriteString(quoteIdentifierPostgres(col.Name))
		b.WriteString(" ")
		b.WriteString(col.DataType)
		if strings.EqualFold(col.IsNullable, "NO") {
			b.WriteString(" NOT NULL")
		}
		if col.DefaultVal.Valid {
			b.WriteString(" DEFAULT ")
			b.WriteString(col.DefaultVal.String)
		}
		if i < len(columns)-1 || len(primaryKeys) > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	if len(primaryKeys) > 0 {
		b.WriteString("  PRIMARY KEY (")
		b.WriteString(strings.Join(primaryKeys, ", "))
		b.WriteString(")\n")
	}

	b.WriteString(");")

	return b.String(), nil
}

func quoteIdentifierPostgres(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
