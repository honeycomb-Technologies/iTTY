package tmux

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrSessionNotFound indicates that the requested tmux session does not exist.
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionExists indicates that the requested tmux session already exists.
var ErrSessionExists = errors.New("session already exists")

// ErrInvalidSessionName indicates that a session name is empty or unsafe.
var ErrInvalidSessionName = errors.New("invalid session name")

// Session represents a tmux session with its metadata.
type Session struct {
	Name         string    `json:"name"`
	Windows      int       `json:"windows"`
	Created      time.Time `json:"created"`
	Attached     bool      `json:"attached"`
	LastPaneCmd  string    `json:"lastPaneCommand,omitempty"`
	LastPanePath string    `json:"lastPanePath,omitempty"`
}

// Window represents a tmux window within a session.
type Window struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Panes  []Pane `json:"panes"`
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

// NewSession creates a new detached tmux session with the given name.
// Returns the newly created session's detail.
func (c *Client) NewSession(ctx context.Context, name string) (*SessionDetail, error) {
	if err := validateSessionName(name); err != nil {
		return nil, err
	}

	_, err := c.run(ctx, "new-session", "-d", "-s", name)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate session") {
			return nil, fmt.Errorf("%w: %s", ErrSessionExists, name)
		}
		return nil, fmt.Errorf("creating session %q: %w", name, err)
	}

	return c.GetSession(ctx, name)
}

// ListSessions returns all tmux sessions with metadata.
// Returns an empty slice (not nil) if no sessions exist.
func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	format := "#{session_name}|#{session_windows}|#{session_created}|#{session_attached}|#{pane_current_command}|#{pane_current_path}"
	out, err := c.run(ctx, "list-sessions", "-F", format)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []Session{}, nil
	}

	sessions := make([]Session, 0, strings.Count(out, "\n")+1)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		session, err := parseSessionLine(line)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetSession returns detailed information about a specific session,
// including its windows and panes.
func (c *Client) GetSession(ctx context.Context, name string) (*SessionDetail, error) {
	format := "#{session_name}|#{session_windows}|#{session_created}|#{session_attached}|#{pane_current_command}|#{pane_current_path}"
	out, err := c.run(ctx, "display-message", "-p", "-t", name, format)
	if err != nil {
		if strings.Contains(err.Error(), "can't find session") {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, name)
		}
		return nil, err
	}
	if out == "" {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, name)
	}

	session, err := parseSessionLine(strings.Split(out, "\n")[0])
	if err != nil {
		return nil, err
	}

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
	winFormat := "#{window_index}|#{window_name}|#{window_active}"
	winOut, err := c.run(ctx, "list-windows", "-t", sessionName, "-F", winFormat)
	if err != nil {
		return nil, err
	}
	if winOut == "" {
		return []Window{}, nil
	}

	windows := make([]Window, 0, strings.Count(winOut, "\n")+1)
	for _, line := range strings.Split(winOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		window, err := parseWindowLine(line)
		if err != nil {
			return nil, err
		}

		panes, err := c.listPanes(ctx, sessionName, window.Index)
		if err != nil {
			return nil, err
		}
		window.Panes = panes
		windows = append(windows, window)
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

	panes := make([]Pane, 0, strings.Count(out, "\n")+1)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		pane, err := parsePaneLine(line)
		if err != nil {
			return nil, err
		}
		panes = append(panes, pane)
	}

	return panes, nil
}

// parseSessionLine parses a single line of tmux list-sessions output.
func parseSessionLine(line string) (Session, error) {
	parts := strings.SplitN(line, "|", 6)
	if len(parts) < 4 {
		return Session{}, fmt.Errorf("malformed session line: %q", line)
	}

	windows, err := strconv.Atoi(parts[1])
	if err != nil {
		return Session{}, fmt.Errorf("invalid session windows in %q: %w", line, err)
	}

	createdEpoch, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return Session{}, fmt.Errorf("invalid session created time in %q: %w", line, err)
	}

	session := Session{
		Name:     parts[0],
		Windows:  windows,
		Created:  time.Unix(createdEpoch, 0),
		Attached: parts[3] != "0",
	}
	if len(parts) > 4 {
		session.LastPaneCmd = parts[4]
	}
	if len(parts) > 5 {
		session.LastPanePath = parts[5]
	}

	return session, nil
}

func validateSessionName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: session name is required", ErrInvalidSessionName)
	}
	if trimmed := strings.TrimSpace(name); trimmed != name {
		return fmt.Errorf("%w: session name cannot have leading or trailing whitespace", ErrInvalidSessionName)
	}
	if len(name) > 64 {
		return fmt.Errorf("%w: session name must be 64 characters or fewer", ErrInvalidSessionName)
	}

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return fmt.Errorf("%w: session name may only contain letters, numbers, dot, underscore, and hyphen", ErrInvalidSessionName)
		}
	}

	return nil
}

func parseWindowLine(line string) (Window, error) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) != 3 {
		return Window{}, fmt.Errorf("malformed window line: %q", line)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return Window{}, fmt.Errorf("invalid window index in %q: %w", line, err)
	}

	return Window{
		Index:  index,
		Name:   parts[1],
		Active: parts[2] == "1",
		Panes:  []Pane{},
	}, nil
}

func parsePaneLine(line string) (Pane, error) {
	parts := strings.SplitN(line, "|", 7)
	if len(parts) != 7 {
		return Pane{}, fmt.Errorf("malformed pane line: %q", line)
	}

	index, err := strconv.Atoi(parts[1])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid pane index in %q: %w", line, err)
	}

	width, err := strconv.Atoi(parts[5])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid pane width in %q: %w", line, err)
	}

	height, err := strconv.Atoi(parts[6])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid pane height in %q: %w", line, err)
	}

	return Pane{
		ID:      parts[0],
		Index:   index,
		Active:  parts[2] == "1",
		Command: parts[3],
		Path:    parts[4],
		Width:   width,
		Height:  height,
	}, nil
}
