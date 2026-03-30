package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

// controlMessage is a JSON message for non-data control commands
// (resize, ping, error).
type controlMessage struct {
	Type    string `json:"type"`
	Cols    int    `json:"cols,omitempty"`
	Rows    int    `json:"rows,omitempty"`
	Message string `json:"message,omitempty"`
}

// handleTerminalWebSocket upgrades to WebSocket and proxies terminal I/O
// for a specific tmux session. The daemon spawns `tmux -CC attach` in a
// PTY and relays bytes bidirectionally. No SSH credentials needed.
func (s *Server) handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionName := r.URL.Query().Get("session")
	if sessionName == "" {
		writeError(w, http.StatusBadRequest, "session query parameter is required")
		return
	}

	if s.tmuxClient == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux not available")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Tailnet-only access
	})
	if err != nil {
		log.Printf("terminal-ws: accept: %v", err)
		return
	}
	defer conn.CloseNow()

	ctx := r.Context()

	// Spawn tmux -CC attach in a PTY
	term, err := s.tmuxClient.AttachInteractive(ctx, sessionName)
	if err != nil {
		log.Printf("terminal-ws: attach %q: %v", sessionName, err)
		conn.Close(websocket.StatusInternalError, err.Error())
		return
	}
	defer term.Close()

	log.Printf("terminal-ws: client attached to session %q", sessionName)

	// Error channel to coordinate shutdown
	done := make(chan struct{})

	// PTY → WebSocket (terminal output)
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := term.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				if err := conn.Write(ctx, websocket.MessageBinary, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	// WebSocket → PTY (user input + control messages)
	go func() {
		for {
			msgType, data, err := conn.Read(ctx)
			if err != nil {
				term.Close()
				return
			}

			switch msgType {
			case websocket.MessageBinary:
				// Raw terminal input
				if _, err := term.Write(data); err != nil {
					term.Close()
					return
				}

			case websocket.MessageText:
				// Control message (resize, ping, etc.)
				var msg controlMessage
				if err := json.Unmarshal(data, &msg); err != nil {
					continue
				}
				switch msg.Type {
				case "resize":
					if msg.Cols > 0 && msg.Rows > 0 {
						term.Resize(uint16(msg.Rows), uint16(msg.Cols))
					}
				case "ping":
					conn.Write(ctx, websocket.MessageText, []byte(`{"type":"pong"}`))
				}
			}
		}
	}()

	// Wait for PTY to close (session ended or client disconnected)
	<-done
	conn.Close(websocket.StatusNormalClosure, "session ended")
	log.Printf("terminal-ws: client detached from session %q", sessionName)
}
