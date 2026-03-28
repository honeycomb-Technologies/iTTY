package tmux

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Session represents a tmux session with its metadata.
type Session struct {
	Name      string    `json:"name"`
	Windows   int       `json:"windows"`
	Created   time.Time `json:"created"`
	Attached  bool      `json:"attached"`
	LastPaneCmd string  `json:"lastPaneCommand,omitempty"`
	LastPanePath string `json:"lastPanePath,omitempty"`
}

// Window represents a tmux window within a session.
type Window struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Panes   []Pane `json:"panes"`
}

// Pane represents a tmux pane within a window.
type Pane struct {
	ID      string `json:"id"`
	Index   int    `json:"index"`
	Active  bool   `json:"active"`
	Command string `json:"command"`
	Path    string `json:"path"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// SessionDetail provides full detail for a single session including windows and panes.
type SessionDetail struct {
	Session
	WindowList []Window `json:"windowList"`
}

// ListSessions returns all tmux sessions with metadata.
// Returns an empty slice (not nil) if no sessions exist.
func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	// Format: name|windows|created_epoch|attached|pane_command|pane_path
	format := "#{session_name}|#{session_windows}|#{session_created}|#{session_attached}|#{pane_current_command}|#{pane_current_path}"
	out, err := c.run(ctx, "list-sessions", "-F", format)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []Session{}, nil
	}

	var sessions []Session
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		s, err := parseSessionLine(line)
		if err != nil {
			continue // skip malformed lines
		}
		sessions = append(sessions, s)
	}

	if sessions == nil {
		sessions = []Session{}
	}
	return sessions, nil
}

// GetSession returns detailed information about a specific session,
// including its windows and panes.
func (c *Client) GetSession(ctx context.Context, name string) (*SessionDetail, error) {
	// First get session info
	format := "#{session_name}|#{session_windows}|#{session_created}|#{session_attached}|#{pane_current_command}|#{pane_current_path}"
	out, err := c.run(ctx, "list-sessions", "-F", format, "-f", "#{==:#{session_name},"+name+"}")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, fmt.Errorf("session %q not found", name)
	}

	session, err := parseSessionLine(strings.Split(out, "\n")[0])
	if err != nil {
		return nil, err
	}

	// Get windows
	windows, err := c.listWindows(ctx, name)
	if err != nil {
		return nil, err
	}

	return &SessionDetail{
		Session:    session,
		WindowList: windows,
	}, nil
}

// listWindows returns all windows in a session with their panes.
func (c *Client) listWindows(ctx context.Context, sessionName string) ([]Window, error) {
	// Get windows
	winFormat := "#{window_index}|#{window_name}|#{window_active}"
	winOut, err := c.run(ctx, "list-windows", "-t", sessionName, "-F", winFormat)
	if err != nil {
		return nil, err
	}
	if winOut == "" {
		return []Window{}, nil
	}

	var windows []Window
	for _, line := range strings.Split(winOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		idx, _ := strconv.Atoi(parts[0])
		w := Window{
			Index:  idx,
			Name:   parts[1],
			Active: parts[2] == "1",
		}

		// Get panes for this window
		panes, err := c.listPanes(ctx, sessionName, idx)
		if err == nil {
			w.Panes = panes
		}
		windows = append(windows, w)
	}

	return windows, nil
}

// listPanes returns all panes in a specific window.
func (c *Client) listPanes(ctx context.Context, sessionName string, windowIndex int) ([]Pane, error) {
	target := fmt.Sprintf("%s:%d", sessionName, windowIndex)
	paneFormat := "#{pane_id}|#{pane_index}|#{pane_active}|#{pane_current_command}|#{pane_current_path}|#{pane_width}|#{pane_height}"
	out, err := c.run(ctx, "list-panes", "-t", target, "-F", paneFormat)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []Pane{}, nil
	}

	var panes []Pane
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 7 {
			continue
		}
		idx, _ := strconv.Atoi(parts[1])
		w, _ := strconv.Atoi(parts[5])
		h, _ := strconv.Atoi(parts[6])
		panes = append(panes, Pane{
			ID:      parts[0],
			Index:   idx,
			Active:  parts[2] == "1",
			Command: parts[3],
			Path:    parts[4],
			Width:   w,
			Height:  h,
		})
	}

	return panes, nil
}

// parseSessionLine parses a single line of tmux list-sessions output.
func parseSessionLine(line string) (Session, error) {
	parts := strings.SplitN(line, "|", 6)
	if len(parts) < 4 {
		return Session{}, fmt.Errorf("malformed session line: %q", line)
	}

	windows, _ := strconv.Atoi(parts[1])
	createdEpoch, _ := strconv.ParseInt(parts[2], 10, 64)
	attached := parts[3] != "0"

	s := Session{
		Name:     parts[0],
		Windows:  windows,
		Created:  time.Unix(createdEpoch, 0),
		Attached: attached,
	}
	if len(parts) > 4 {
		s.LastPaneCmd = parts[4]
	}
	if len(parts) > 5 {
		s.LastPanePath = parts[5]
	}
	return s, nil
}

