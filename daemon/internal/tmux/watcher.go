package tmux

import (
	"context"
	"log"
	"time"
)

// EventType identifies a session lifecycle event.
type EventType string

const (
	// EventSessionCreated is emitted when a new tmux session appears.
	EventSessionCreated EventType = "session.created"

	// EventSessionClosed is emitted when a tmux session is destroyed.
	EventSessionClosed EventType = "session.closed"

	// EventSessionUpdated is emitted when a session's window/pane count or
	// attached state changes.
	EventSessionUpdated EventType = "session.updated"
)

// SessionEvent describes a change to the tmux session list.
type SessionEvent struct {
	Type    EventType `json:"type"`
	Session Session   `json:"session"`
}

// WatchSessions polls tmux at the given interval and emits events
// whenever sessions are created, destroyed, or modified. The returned
// channel is closed when ctx is cancelled. Callers should drain the
// channel to avoid blocking the watcher goroutine.
func (c *Client) WatchSessions(ctx context.Context, interval time.Duration) <-chan SessionEvent {
	ch := make(chan SessionEvent, 32)

	go func() {
		defer close(ch)

		prev := make(map[string]Session)

		for {
			sessions, err := c.ListSessions(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("watcher: list sessions: %v", err)
			} else {
				curr := make(map[string]Session, len(sessions))
				for _, s := range sessions {
					curr[s.Name] = s
				}

				// Detect new and updated sessions.
				for name, s := range curr {
					old, existed := prev[name]
					if !existed {
						emit(ctx, ch, SessionEvent{Type: EventSessionCreated, Session: s})
					} else if sessionChanged(old, s) {
						emit(ctx, ch, SessionEvent{Type: EventSessionUpdated, Session: s})
					}
				}

				// Detect closed sessions.
				for name, s := range prev {
					if _, exists := curr[name]; !exists {
						emit(ctx, ch, SessionEvent{Type: EventSessionClosed, Session: s})
					}
				}

				prev = curr
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()

	return ch
}

// sessionChanged returns true if the session metadata differs in a way
// that warrants an update event.
func sessionChanged(a, b Session) bool {
	return a.Windows != b.Windows ||
		a.Attached != b.Attached ||
		a.LastPaneCmd != b.LastPaneCmd
}

// emit sends an event to the channel without blocking if the channel is
// full or the context is cancelled.
func emit(ctx context.Context, ch chan<- SessionEvent, event SessionEvent) {
	select {
	case ch <- event:
	case <-ctx.Done():
	default:
		// Channel full — drop event rather than blocking the watcher.
	}
}
