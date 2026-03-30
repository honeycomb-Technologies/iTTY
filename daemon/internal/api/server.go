// Package api provides the HTTP server for the iTTY daemon.
package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/apns"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/config"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/platform"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/shell"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tailscale"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tmux"
)

type tmuxService interface {
	IsInstalled(context.Context) bool
	Version(context.Context) (string, error)
	ListSessions(context.Context) ([]tmux.Session, error)
	GetSession(context.Context, string) (*tmux.SessionDetail, error)
	NewSession(context.Context, string) (*tmux.SessionDetail, error)
	CaptureSessionDefaultPane(context.Context, string) (string, error)
}

type shellManager interface {
	Detect() (*shell.ShellInfo, error)
	Configure(*shell.ShellInfo) error
	Unconfigure(*shell.ShellInfo) error
}

type tailscalePeers interface {
	Peers(context.Context) ([]tailscale.Peer, error)
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
	tmuxClient  *tmux.Client
	tailscale   tailscalePeers
	config      *config.Config
	windows     platform.WindowDiscoverer
	shell       shellManager
	configStore configStore
	wsHub       *wsHub
	apnsSender  *apns.Sender
	deviceStore *apns.DeviceStore
	mux         *http.ServeMux
	server      *http.Server
}

// NewServer creates a new API server with the default dependencies.
func NewServer(tmuxSvc tmuxService, tmuxCli *tmux.Client, cfg *config.Config, ts tailscalePeers) *Server {
	s := NewServerWithDeps(
		tmuxSvc,
		cfg,
		ts,
		platform.NewWindowDiscoverer(),
		defaultShellManager{},
		defaultConfigStore{},
	)
	s.tmuxClient = tmuxCli
	return s
}

// NewServerWithDeps creates a new API server with explicit dependencies.
func NewServerWithDeps(
	tmuxClient tmuxService,
	cfg *config.Config,
	ts tailscalePeers,
	windows platform.WindowDiscoverer,
	shell shellManager,
	store configStore,
) *Server {
	s := &Server{
		tmux:        tmuxClient,
		tailscale:   ts,
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
	s.mux.HandleFunc("POST /sessions", s.handleCreateSession)
	s.mux.HandleFunc("GET /windows", s.handleGetWindows)
	s.mux.HandleFunc("GET /peers", s.handleGetPeers)
	s.mux.HandleFunc("GET /ws", s.handleWebSocket)
	s.mux.HandleFunc("GET /ws/terminal", s.handleTerminalWebSocket)
	s.mux.HandleFunc("POST /devices", s.handleRegisterDevice)
	s.mux.HandleFunc("DELETE /devices/{token}", s.handleUnregisterDevice)
}

// ConfigureAPNs sets up push notification support. Safe to call with nil sender.
// The device store is always created so that device registration works even
// before APNs credentials are configured — tokens are stored and ready to
// use once a sender is provided.
func (s *Server) ConfigureAPNs(sender *apns.Sender) {
	s.apnsSender = sender
	if s.deviceStore == nil {
		s.deviceStore = apns.NewDeviceStore()
	}
}

// APNsSender returns the APNs sender, or nil if not configured.
func (s *Server) APNsSender() *apns.Sender { return s.apnsSender }

// DeviceStore returns the device token store, or nil if APNs is not configured.
func (s *Server) DeviceStore() *apns.DeviceStore { return s.deviceStore }

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
