package tmux

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// TerminalSession represents an interactive tmux control mode session
// attached via a PTY. Bytes flow bidirectionally — read for terminal
// output, write for user input.
type TerminalSession struct {
	cmd     *exec.Cmd
	ptmx    *os.File
	session string
	mu      sync.Mutex
	closed  bool
}

// AttachInteractive spawns `tmux -CC attach -t <session>` in a PTY
// and returns a TerminalSession for bidirectional I/O. The caller
// owns the session and must call Close when done.
func (c *Client) AttachInteractive(ctx context.Context, sessionName string) (*TerminalSession, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name is required")
	}

	cmd := exec.CommandContext(ctx, c.TmuxPath, "-CC", "attach-session", "-t", sessionName)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("starting tmux attach: %w", err)
	}

	return &TerminalSession{
		cmd:     cmd,
		ptmx:    ptmx,
		session: sessionName,
	}, nil
}

// Read reads terminal output from the PTY.
func (ts *TerminalSession) Read(b []byte) (int, error) {
	return ts.ptmx.Read(b)
}

// Write sends user input to the PTY.
func (ts *TerminalSession) Write(b []byte) (int, error) {
	return ts.ptmx.Write(b)
}

// Resize changes the PTY window size.
func (ts *TerminalSession) Resize(rows, cols uint16) error {
	return pty.Setsize(ts.ptmx, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// SessionName returns the tmux session this terminal is attached to.
func (ts *TerminalSession) SessionName() string {
	return ts.session
}

// Close terminates the tmux attach process and closes the PTY.
func (ts *TerminalSession) Close() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closed {
		return nil
	}
	ts.closed = true

	// Close the PTY — this signals EOF to the process.
	if ts.ptmx != nil {
		ts.ptmx.Close()
	}

	// Wait for the process to exit (don't leave zombies).
	if ts.cmd != nil && ts.cmd.Process != nil {
		ts.cmd.Wait()
	}

	return nil
}

// Wait blocks until the tmux process exits and returns any error.
func (ts *TerminalSession) Wait() error {
	if ts.cmd == nil {
		return nil
	}
	return ts.cmd.Wait()
}

// WriteTo copies all terminal output to the given writer until
// the session ends. Useful for piping to a WebSocket.
func (ts *TerminalSession) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, ts.ptmx)
}
