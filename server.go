package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AgentServer struct {
	pool   map[string]*sql.DB
	mu     sync.RWMutex
	logger *log.Logger
}

// Close shuts down all pooled database connections.
func (s *AgentServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, db := range s.pool {
		db.Close()
		s.logger.Printf("Closed connection to server %q", name)
	}
}

// getConn returns the pooled *sql.DB for serverName, or an error if not found.
func (s *AgentServer) getConn(serverName string) (*sql.DB, error) {
	s.mu.RLock()
	db := s.pool[serverName]
	s.mu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("no connection for server %q: check server name and configuration", serverName)
	}
	return db, nil
}

// initPool opens and pings a connection for every server in databaseServersConfig.
// Servers that cannot be reached at startup are skipped and logged.
func (s *AgentServer) initPool(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pool = make(map[string]*sql.DB)
	for serverName, config := range databaseServersConfig {
		port := config.Port
		if port == "" {
			port = "3306"
		}
		// No database in DSN; queries must qualify table names as database.table.
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", config.User, config.Password, config.Host, port)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			s.logger.Printf("Failed to open connection for server %q: %v", serverName, err)
			continue
		}
		if err := db.PingContext(ctx); err != nil {
			s.logger.Printf("Failed to ping server %q: %v — skipping", serverName, err)
			db.Close()
			continue
		}
		s.pool[serverName] = db
		s.logger.Printf("Connected to server %q (%s:%s)", serverName, config.Host, port)
	}
}

// RegisterTools wires up all the separated tool handlers to the MCP server
func (s *AgentServer) RegisterTools(server *mcp.Server, logger *log.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_available_servers",
		Description: "Lists all available database servers.",
	}, HandleListAvailableServers)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_all_databases",
		Description: "Lists all databases on the configured server.",
	}, HandleListAllDatabases)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "execute_select_query",
		Description: "Executes a read-only SELECT statement against a specific server. USE THIS STRICTLY FOR FETCHING DATA. Do NOT use this tool for INSERT, UPDATE, DELETE, or schema modifications. Because no default database is selected, you MUST qualify all table references as database_name.table_name in your SQL (e.g. SELECT * FROM mydb.users). IMPORTANT: You MUST provide a clear explanation in the 'explanation' field detailing the user's intent before providing the SQL.",
	}, HandleExecuteSelectQuery)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "inspect_single_table_schema",
		Description: "Retrieves the CREATE TABLE DDL for ONE specific table. Use this to learn the exact column names and data types BEFORE writing a SELECT or INSERT query. You MUST know the exact table name and database name to use this.",
	}, HandleInspectSingleTable)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "execute_write_statement",
		Description: "Executes a raw SQL write operation (INSERT, UPDATE, CREATE, ALTER) against a specific server. USE THIS STRICTLY FOR MODIFYING DATA OR SCHEMA. Do NOT use this tool for fetching data; use execute_select_query instead. Because no default database is selected, you MUST qualify all table references as database_name.table_name in your SQL (e.g. INSERT INTO mydb.users ...). Always verify table schema first before inserting or updating. IMPORTANT: You MUST provide a clear explanation in the 'explanation' field detailing the user's intent before providing the SQL.",
	}, HandleExecuteWriteStatement)
}

func (s *AgentServer) LogIncomingRequest(req *mcp.CallToolRequest) {
	requestForLog := struct {
		Params *mcp.CallToolParamsRaw `json:"params"`
	}{
		Params: req.Params,
	}

	reqBytes, err := json.MarshalIndent(requestForLog, "", "  ")
	if err != nil {
		s.logger.Printf("Warning: Failed to marshal incoming request for logging: %v\n", err)
	} else {
		s.logger.Printf("Raw request received from Copilot:\n%s\n", string(reqBytes))
	}
}

func (s *AgentServer) LogOutgoingResponse(payload any) {
	payloadBytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		s.logger.Printf("Warning: Failed to marshal output: %v\n", err)
		return
	}
	s.logger.Printf("Data returned to Copilot:\n%s\n", string(payloadBytes))
}
