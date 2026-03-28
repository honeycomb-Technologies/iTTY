package api

import (
	"net/http"
	"runtime"
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

// handleGetSession returns detailed info for a single session.
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "session name required")
		return
	}

	detail, err := s.tmux.GetSession(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
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
	// TODO: Parse request body for {"enabled": true/false}
	// TODO: Call shell.Configure() or shell.Unconfigure()
	writeError(w, http.StatusNotImplemented, "not yet implemented")
}
