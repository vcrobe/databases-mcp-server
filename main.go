package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var logger *log.Logger
var app *AgentServer

func main() {
	healthcheck := flag.Bool("healthcheck", false, "run as a health check client against the running server and exit")
	flag.Parse()

	// When invoked as the Docker HEALTHCHECK CMD, perform a single HTTP probe
	// and exit immediately — the main server process is not started.
	if *healthcheck {
		runHealthcheck()
	}

	var logFile *os.File
	app, logFile = createAgentServer()
	defer logFile.Close()
	defer app.Close()

	// Initialize the MCP Server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "databases-mcp-server",
		Version: "1.0.0",
	}, nil)

	// IMPORTANT: Do not use fmt.Println or log to stdout, as it corrupts the JSON-RPC stream.
	log.SetOutput(os.Stderr)

	// ctx is cancelled on SIGTERM or SIGINT, triggering graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Read database server configurations from environment variables and open
	// a pooled connection for every configured server.
	if err := readDatabaseServersConfig(); err != nil {
		logger.Fatalf("invalid database configuration: %v", err)
	}
	app.initPool(ctx)

	// Call the centralized registry
	app.RegisterTools(server, logger)

	// Start the MCP server with the chosen transport
	startMCPServer(ctx, server)
}

func startMCPServer(ctx context.Context, server *mcp.Server) {
	transport := strings.ToLower(strings.TrimSpace(os.Getenv("MCP_TRANSPORT")))

	switch transport {
	case "", "stdio":
		// Copilot will communicate with this process over stdin/stdout.
		wiretapStdio(logger)
		log.Println("Starting databases-mcp-server on stdio...")
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatalf("Server error: %v", err)
		}
		logger.Println("Client disconnected: stdio closed")
		log.Println("Client disconnected: stdio closed")

	case "streamable-http", "http":
		addr := os.Getenv("MCP_HTTP_ADDR")
		if addr == "" {
			addr = ":8080"
		}

		mux := http.NewServeMux()
		mux.Handle("/", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
		mux.HandleFunc("/health", healthHandler)

		srv := &http.Server{
			Addr:    addr,
			Handler: loggingMiddleware(mux),
		}
		go gracefulShutdown(ctx, srv)

		logger.Printf("Starting databases-mcp-server on streamable HTTP at '%s'", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server error: %v", err)
		}

	case "sse":
		addr := os.Getenv("MCP_HTTP_ADDR")
		if addr == "" {
			addr = ":8080"
		}

		mux := http.NewServeMux()
		mux.Handle("/", mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return server }, nil))
		mux.HandleFunc("/health", healthHandler)

		srv := &http.Server{
			Addr:    addr,
			Handler: loggingMiddleware(mux),
		}
		go gracefulShutdown(ctx, srv)

		logger.Printf("Starting databases-mcp-server on legacy SSE at '%s'", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server error: %v", err)
		}

	default:
		logger.Fatalf("unsupported MCP_TRANSPORT %q (expected: stdio, streamable-http, or sse)", transport)
	}
}

// gracefulShutdown waits for ctx to be cancelled then gives in-flight requests
// up to 10 seconds to finish before forcefully closing the server.
func gracefulShutdown(ctx context.Context, srv *http.Server) {
	<-ctx.Done()
	logger.Println("Shutdown signal received, draining connections...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("Graceful shutdown error: %v", err)
	}
}

func createAgentServer() (*AgentServer, *os.File) {
	// Set up dedicated file logging FIRST
	logFile, err := os.OpenFile("mcp-audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// It is safe to write to stderr if we fail to open the file
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}

	// Create a custom logger that writes exclusively to this file
	logger = log.New(logFile, "[MCP-DB] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lmsgprefix)

	return &AgentServer{
		logger: logger,
	}, logFile
}
