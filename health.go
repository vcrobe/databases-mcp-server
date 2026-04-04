package main

import (
	"encoding/json"
	"net/http"
	"os"
)

type healthResponse struct {
	Status  string            `json:"status"`
	Servers map[string]string `json:"servers"`
}

// healthHandler serves GET /health.
// Returns 200 with status "ok" when all server connections are healthy, or
// 503 with status "degraded" when any configured server cannot be pinged.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	app.mu.RLock()
	servers := make(map[string]string, len(app.pool))
	for name, db := range app.pool {
		if err := db.PingContext(r.Context()); err != nil {
			logger.Printf("Health check: ping failed for server %q: %v", name, err)
			servers[name] = "unavailable"
		} else {
			servers[name] = "ok"
		}
	}
	app.mu.RUnlock()

	status := "ok"
	for _, s := range servers {
		if s != "ok" {
			status = "degraded"
			break
		}
	}

	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(healthResponse{ //nolint:errcheck
		Status:  status,
		Servers: servers,
	})
}

// runHealthcheck performs a single GET /health against the running server and
// exits 0 (healthy) or 1 (unhealthy / unreachable).
// Used as the Docker HEALTHCHECK CMD: databases-mcp-server -healthcheck
func runHealthcheck() {
	addr := os.Getenv("MCP_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	// ":8080" → "127.0.0.1:8080"
	if len(addr) > 0 && addr[0] == ':' {
		addr = "127.0.0.1" + addr
	}

	resp, err := http.Get("http://" + addr + "/health") //nolint:noctx
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}
