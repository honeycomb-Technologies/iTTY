// Package tmux provides a client for interacting with tmux via command execution.
// It handles listing sessions, windows, and panes, capturing pane content,
// and parsing tmux's structured output formats.
package tmux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client executes tmux commands and parses their output.
// All methods are safe for concurrent use — each call spawns
// a new tmux process with no shared state.
type Client struct {
	// TmuxPath is the path to the tmux binary. Defaults to "tmux".
	TmuxPath string

	// CommandTimeout is the maximum duration for a single tmux command.
	// Defaults to 5 seconds.
	CommandTimeout time.Duration
}

// NewClient creates a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		TmuxPath:       "tmux",
		CommandTimeout: 5 * time.Second,
	}
}

// run executes a tmux command and returns its stdout.
// Returns an error if the command fails or times out.
func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	timeout := c.CommandTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.TmuxPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// tmux returns exit code 1 with "no server running" when no sessions exist
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "no server running") ||
			strings.Contains(stderrStr, "no sessions") ||
			strings.Contains(stderrStr, "no current session") ||
			strings.Contains(stderrStr, "error connecting to") {
			return "", nil
		}
		return "", fmt.Errorf("tmux %s: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsInstalled checks whether tmux is available on the system.
func (c *Client) IsInstalled(ctx context.Context) bool {
	_, err := exec.LookPath(c.TmuxPath)
	return err == nil
}

// IsRunning checks whether a tmux server is currently running.
func (c *Client) IsRunning(ctx context.Context) bool {
	_, err := c.run(ctx, "list-sessions", "-F", "#{session_name}")
	return err == nil
}

// Version returns the tmux version string (e.g., "3.6a").
func (c *Client) Version(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "-V")
	if err != nil {
		return "", err
	}
	// Output format: "tmux 3.6a"
	return strings.TrimPrefix(out, "tmux "), nil
}
