package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tmux"
)

// wsHub manages WebSocket client connections and broadcasts tmux
// session events to all connected clients.
type wsHub struct {
	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
}

func newWSHub() *wsHub {
	return &wsHub{
		clients: make(map[*wsClient]struct{}),
	}
}

func (h *wsHub) addClient(c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *wsHub) removeClient(c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	close(c.send)
}

func (h *wsHub) broadcast(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// Client too slow — skip this event.
		}
	}
}

func (h *wsHub) clientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// StartSessionWatcher begins watching tmux for session changes and
// broadcasting events to all connected WebSocket clients. Returns a
// cancel function to stop the watcher.
func (s *Server) StartSessionWatcher(ctx context.Context, tmuxClient *tmux.Client, interval time.Duration) context.CancelFunc {
	s.wsHub = newWSHub()

	watchCtx, cancel := context.WithCancel(ctx)

	events := tmuxClient.WatchSessions(watchCtx, interval)

	go func() {
		for event := range events {
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("websocket: marshal event: %v", err)
				continue
			}
			s.wsHub.broadcast(data)
		}
	}()

	return cancel
}

// handleWebSocket upgrades an HTTP connection to WebSocket and streams
// tmux session events to the client.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.wsHub == nil {
		writeError(w, http.StatusServiceUnavailable, "websocket not available")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin (tailnet-only).
	})
	if err != nil {
		log.Printf("websocket: accept: %v", err)
		return
	}

	client := &wsClient{
		conn: conn,
		send: make(chan []byte, 64),
	}

	s.wsHub.addClient(client)
	log.Printf("websocket: client connected (%d total)", s.wsHub.clientCount())

	ctx := r.Context()

	// Writer goroutine: sends events and heartbeat pings.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-client.send:
				if !ok {
					return
				}
				if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
					return
				}
			case <-ticker.C:
				if err := conn.Ping(ctx); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Reader goroutine: reads and discards client messages (keeps connection alive).
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	s.wsHub.removeClient(client)
	conn.Close(websocket.StatusNormalClosure, "")
	log.Printf("websocket: client disconnected (%d remaining)", s.wsHub.clientCount())
}
