package mcp

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"testing"
	"time"
)

// TestServe serves the endpoint on MCP_SERVE for the official conformance
// suite or MCP Inspector, with the administrator's token added to every
// request that carries none, until interrupted.
func TestServe(t *testing.T) {
	addr := os.Getenv("MCP_SERVE")
	if addr == "" {
		t.Skip("set MCP_SERVE=127.0.0.1:<port> to serve the endpoint")
	}
	listener, err := net.Listen("tcp", addr)
	must(t, err)
	f := newFixtureOn(t, listener)
	token := f.token(t)
	f.front = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			next.ServeHTTP(w, r)
		})
	}
	f.entry(t, "nuage", "error", "upload failed", time.Minute, map[string]any{"request_id": "req-1"})
	t.Logf("serving %s", f.resource)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
