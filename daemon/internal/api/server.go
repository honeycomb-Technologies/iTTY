// Package api provides the HTTP and WebSocket server for the iTTY daemon.
// It exposes tmux session information and real-time updates to the iTTY iOS app.
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/config"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tmux"
)

// Server is the iTTY daemon HTTP server.
type Server struct {
	tmux   *tmux.Client
	config *config.Config
	mux    *http.ServeMux
	server *http.Server
}

// NewServer creates a new API server with the given dependencies.
func NewServer(tmuxClient *tmux.Client, cfg *config.Config) *Server {
	s := &Server{
		tmux:   tmuxClient,
		config: cfg,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// registerRoutes sets up all HTTP route handlers.
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /sessions", s.handleListSessions)
	s.mux.HandleFunc("GET /sessions/{name}", s.handleGetSession)
	s.mux.HandleFunc("GET /sessions/{name}/content", s.handleCaptureContent)
	s.mux.HandleFunc("GET /config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /config/auto", s.handleSetAutoWrap)
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.config.ListenAddr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("iTTY daemon listening on %s", s.config.ListenAddr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
