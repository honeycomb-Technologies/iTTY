package api

import (
	"encoding/json"
	"net/http"
)

type registerDeviceRequest struct {
	Token string `json:"token"`
}

// handleRegisterDevice registers an APNs device token for push notifications.
func (s *Server) handleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	if s.deviceStore == nil {
		writeError(w, http.StatusServiceUnavailable, "push notifications not configured")
		return
	}

	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "device token is required")
		return
	}

	s.deviceStore.Register(req.Token)
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

// handleUnregisterDevice removes an APNs device token.
func (s *Server) handleUnregisterDevice(w http.ResponseWriter, r *http.Request) {
	if s.deviceStore == nil {
		writeError(w, http.StatusServiceUnavailable, "push notifications not configured")
		return
	}

	token := r.PathValue("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "device token is required")
		return
	}

	s.deviceStore.Unregister(token)
	writeJSON(w, http.StatusOK, map[string]string{"status": "unregistered"})
}
