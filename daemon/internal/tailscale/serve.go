// Package tailscale handles integration with the Tailscale CLI,
// primarily auto-configuring `tailscale serve` to expose the daemon
// on the user's private tailnet.
package tailscale

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Client wraps the Tailscale CLI for daemon integration.
type Client struct {
	// BinPath is the path to the tailscale binary. Defaults to "tailscale".
	BinPath string

	// CommandTimeout is the maximum duration for a single CLI command.
	CommandTimeout time.Duration
}

// NewClient creates a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		BinPath:        "tailscale",
		CommandTimeout: 10 * time.Second,
	}
}

// IsInstalled checks whether the tailscale CLI is available.
func (c *Client) IsInstalled() bool {
	_, err := exec.LookPath(c.BinPath)
	return err == nil
}

// IsRunning checks whether Tailscale is connected to a tailnet.
func (c *Client) IsRunning(ctx context.Context) bool {
	out, err := c.run(ctx, "status", "--json")
	if err != nil {
		return false
	}
	// If we get JSON output, tailscale is running.
	return strings.Contains(out, `"BackendState"`)
}

// Hostname returns the machine's Tailscale hostname.
func (c *Client) Hostname(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "status", "--self", "--json")
	if err != nil {
		return "", fmt.Errorf("tailscale status: %w", err)
	}
	// Parse hostname from JSON — simplified extraction
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"DNSName"`) {
			// Extract value: "DNSName": "machine.tailnet.ts.net.",
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.Trim(strings.TrimSpace(parts[1]), `",. `)
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("could not determine tailscale hostname")
}

// ServePort configures `tailscale serve` to expose a local port on the tailnet.
// This runs in the background (--bg) so it persists across daemon restarts.
func (c *Client) ServePort(ctx context.Context, port int) error {
	portStr := strconv.Itoa(port)
	_, err := c.run(ctx, "serve", "--bg", portStr)
	if err != nil {
		return fmt.Errorf("tailscale serve --bg %s: %w", portStr, err)
	}
	return nil
}

// ServeReset removes all tailscale serve configurations.
func (c *Client) ServeReset(ctx context.Context) error {
	_, err := c.run(ctx, "serve", "reset")
	if err != nil {
		return fmt.Errorf("tailscale serve reset: %w", err)
	}
	return nil
}

// run executes a tailscale CLI command and returns stdout.
func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	timeout := c.CommandTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.BinPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
