package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tmux"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// healthResponse is returned by GET /health.
type healthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
	TmuxOK   bool   `json:"tmuxInstalled"`
	TmuxVer  string `json:"tmuxVersion,omitempty"`
}

type autoWrapRequest struct {
	Enabled *bool `json:"enabled"`
}

// handleHealth returns daemon status and tmux availability.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := healthResponse{
		Status:   "ok",
		Version:  Version,
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		TmuxOK:   s.tmux.IsInstalled(ctx),
	}

	if resp.TmuxOK {
		if v, err := s.tmux.Version(ctx); err == nil {
			resp.TmuxVer = v
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleListSessions returns all tmux sessions.
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.tmux.ListSessions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

type createSessionRequest struct {
	Name string `json:"name"`
}

// handleCreateSession creates a new detached tmux session.
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "session name is required")
		return
	}

	detail, err := s.tmux.NewSession(r.Context(), req.Name)
	if err != nil {
		switch {
		case errors.Is(err, tmux.ErrInvalidSessionName):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, tmux.ErrSessionExists):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, detail)
}

// handleGetSession returns detailed info for a single session.
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "session name required")
		return
	}

	detail, err := s.tmux.GetSession(r.Context(), name)
	if err != nil {
		if errors.Is(err, tmux.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// handleCaptureContent captures and returns the active pane content
// of a session.
func (s *Server) handleCaptureContent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "session name required")
		return
	}

	content, err := s.tmux.CaptureSessionDefaultPane(r.Context(), name)
	if err != nil {
		if errors.Is(err, tmux.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

// handleGetConfig returns the current daemon configuration.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.config)
}

// handleSetAutoWrap toggles the auto-tmux-wrap shell configuration.
func (s *Server) handleSetAutoWrap(w http.ResponseWriter, r *http.Request) {
	var req autoWrapRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}

	shellInfo, err := s.shell.Detect()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if *req.Enabled {
		err = s.shell.Configure(shellInfo)
	} else {
		err = s.shell.Unconfigure(shellInfo)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	nextConfig := *s.config
	nextConfig.AutoWrap = *req.Enabled
	if err := s.configStore.Save(&nextConfig); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.config.AutoWrap = *req.Enabled
	writeJSON(w, http.StatusOK, s.config)
}

// handleGetPeers returns all devices on the user's tailnet.
func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	if s.tailscale == nil {
		writeError(w, http.StatusServiceUnavailable, "tailscale integration not available")
		return
	}

	peers, err := s.tailscale.Peers(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, peers)
}

// handleGetWindows lists open terminal windows on the desktop.
func (s *Server) handleGetWindows(w http.ResponseWriter, r *http.Request) {
	windows, err := s.windows.ListWindows()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, windows)
}
