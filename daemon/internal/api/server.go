// Package api provides the HTTP server for the iTTY daemon.
package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/config"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/platform"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/shell"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tmux"
)

type tmuxService interface {
	IsInstalled(context.Context) bool
	Version(context.Context) (string, error)
	ListSessions(context.Context) ([]tmux.Session, error)
	GetSession(context.Context, string) (*tmux.SessionDetail, error)
	CaptureSessionDefaultPane(context.Context, string) (string, error)
}

type shellManager interface {
	Detect() (*shell.ShellInfo, error)
	Configure(*shell.ShellInfo) error
	Unconfigure(*shell.ShellInfo) error
}

type configStore interface {
	Save(*config.Config) error
}

type defaultShellManager struct{}

func (defaultShellManager) Detect() (*shell.ShellInfo, error) {
	return shell.Detect()
}

func (defaultShellManager) Configure(info *shell.ShellInfo) error {
	return shell.Configure(info)
}

func (defaultShellManager) Unconfigure(info *shell.ShellInfo) error {
	return shell.Unconfigure(info)
}

type defaultConfigStore struct{}

func (defaultConfigStore) Save(cfg *config.Config) error {
	return config.SaveDefault(cfg)
}

// Server is the iTTY daemon HTTP server.
type Server struct {
	tmux        tmuxService
	config      *config.Config
	windows     platform.WindowDiscoverer
	shell       shellManager
	configStore configStore
	mux         *http.ServeMux
	server      *http.Server
}

// NewServer creates a new API server with the default dependencies.
func NewServer(tmuxClient tmuxService, cfg *config.Config) *Server {
	return NewServerWithDeps(
		tmuxClient,
		cfg,
		platform.NewWindowDiscoverer(),
		defaultShellManager{},
		defaultConfigStore{},
	)
}

// NewServerWithDeps creates a new API server with explicit dependencies.
func NewServerWithDeps(
	tmuxClient tmuxService,
	cfg *config.Config,
	windows platform.WindowDiscoverer,
	shell shellManager,
	store configStore,
) *Server {
	s := &Server{
		tmux:        tmuxClient,
		config:      cfg,
		windows:     windows,
		shell:       shell,
		configStore: store,
		mux:         http.NewServeMux(),
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
	s.mux.HandleFunc("GET /windows", s.handleGetWindows)
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

	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return err
	}

	log.Printf("iTTY daemon listening on %s", listener.Addr())
	return s.server.Serve(listener)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
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
