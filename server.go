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

func (s *AgentServer) getServerConfig(serverName string) (DatabaseServerConfig, error) {
	config, ok := databaseServersConfig[serverName]
	if !ok {
		return DatabaseServerConfig{}, fmt.Errorf("no configuration for server %q: check server name and configuration", serverName)
	}
	return config, nil
}

func dsnForServer(config DatabaseServerConfig, port string) (driver string, dsn string, err error) {
	switch config.Engine {
	case "mysql":
		return "mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", config.User, config.Password, config.Host, port), nil
	case "postgres":
		return "postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", config.Host, port, config.User, config.Password, config.Database), nil
	default:
		return "", "", fmt.Errorf("unsupported database engine %q", config.Engine)
	}
}

// initPool opens and pings a connection for every server in databaseServersConfig.
// Servers that cannot be reached at startup are skipped and logged.
func (s *AgentServer) initPool(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pool = make(map[string]*sql.DB)
	for serverName, config := range databaseServersConfig {
		driver, dsn, err := dsnForServer(config, config.Port)
		if err != nil {
			s.logger.Printf("Invalid server configuration %q: %v", serverName, err)
			continue
		}

		db, err := sql.Open(driver, dsn)
		if err != nil {
			s.logger.Printf("Failed to open %s connection for server %q: %v", config.Engine, serverName, err)
			continue
		}
		if err := db.PingContext(ctx); err != nil {
			s.logger.Printf("Failed to ping %s server %q: %v — skipping", config.Engine, serverName, err)
			db.Close()
			continue
		}
		s.pool[serverName] = db
		s.logger.Printf("Connected to %s server %q (%s:%s)", config.Engine, serverName, config.Host, config.Port)
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
		Description: "Executes a read-only SELECT statement against a specific server. USE THIS STRICTLY FOR FETCHING DATA. Do NOT use this tool for INSERT, UPDATE, DELETE, or schema modifications. Qualify table references for the target engine (MySQL: database_name.table_name, PostgreSQL: schema_name.table_name). IMPORTANT: You MUST provide a clear explanation in the 'explanation' field detailing the user's intent before providing the SQL.",
	}, HandleExecuteSelectQuery)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "inspect_single_table_schema",
		Description: "Retrieves the CREATE TABLE DDL for ONE specific table. Use this to learn the exact column names and data types BEFORE writing a SELECT or INSERT query. You MUST know the exact table name and namespace to use this (MySQL: database name, PostgreSQL: schema name).",
	}, HandleInspectSingleTable)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "execute_write_statement",
		Description: "Executes a raw SQL write operation (INSERT, UPDATE, CREATE, ALTER) against a specific server. USE THIS STRICTLY FOR MODIFYING DATA OR SCHEMA. Do NOT use this tool for fetching data; use execute_select_query instead. Qualify table references for the target engine (MySQL: database_name.table_name, PostgreSQL: schema_name.table_name). Always verify table schema first before inserting or updating. IMPORTANT: You MUST provide a clear explanation in the 'explanation' field detailing the user's intent before providing the SQL.",
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
