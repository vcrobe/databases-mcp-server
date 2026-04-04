package main

import (
	"bytes"
	"io"
	"net/http"
)

// --- Middleware ---

// Note: This middleware is only applied to HTTP transports. For other transports (e.g., stdio, sockets), the MCP server does not provide a built-in way to intercept raw messages, so logging is done at the tool handler level instead.

// loggingMiddleware captures and logs the full JSON-RPC request body for HTTP transports.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		logger.Printf("[HTTP] Incoming JSON-RPC request: %s\n", body)

		r.Body = io.NopCloser(bytes.NewReader(body))

		next.ServeHTTP(w, r)
	})
}
