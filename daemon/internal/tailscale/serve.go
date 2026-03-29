// Package tailscale handles integration with the Tailscale CLI,
// primarily auto-configuring `tailscale serve` to expose the daemon
// on the user's private tailnet.
package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// macOSAppCLI is the path to the Tailscale CLI embedded in the macOS app bundle.
const macOSAppCLI = "/Applications/Tailscale.app/Contents/MacOS/Tailscale"

// NewClient creates a Client with sensible defaults.
// It checks the standard PATH first, then falls back to the macOS app bundle.
func NewClient() *Client {
	bin := "tailscale"
	if _, err := exec.LookPath(bin); err != nil {
		if _, err := os.Stat(macOSAppCLI); err == nil {
			bin = macOSAppCLI
		}
	}
	return &Client{
		BinPath:        bin,
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

	var status struct {
		BackendState string `json:"BackendState"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return false
	}

	return status.BackendState == "Running"
}

// Hostname returns the machine's Tailscale hostname.
func (c *Client) Hostname(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "status", "--self", "--json")
	if err != nil {
		return "", fmt.Errorf("tailscale status: %w", err)
	}
	var status struct {
		DNSName string `json:"DNSName"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return "", fmt.Errorf("decoding tailscale status: %w", err)
	}

	name := strings.TrimSuffix(strings.TrimSpace(status.DNSName), ".")
	if name == "" {
		return "", fmt.Errorf("could not determine tailscale hostname")
	}

	return name, nil
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

// Peer represents a device on the user's tailnet.
type Peer struct {
	ID       string   `json:"id"`
	Hostname string   `json:"hostname"`
	DNSName  string   `json:"dnsName"`
	OS       string   `json:"os"`
	Online   bool     `json:"online"`
	IPs      []string `json:"ips"`
	Self     bool     `json:"self"`
}

// Peers returns all devices on the tailnet, including the local machine.
// Returns an empty slice if Tailscale is not running.
func (c *Client) Peers(ctx context.Context) ([]Peer, error) {
	out, err := c.run(ctx, "status", "--json")
	if err != nil {
		return nil, fmt.Errorf("tailscale status: %w", err)
	}

	var status struct {
		BackendState string `json:"BackendState"`
		Self         struct {
			ID           string   `json:"ID"`
			HostName     string   `json:"HostName"`
			DNSName      string   `json:"DNSName"`
			OS           string   `json:"OS"`
			Online       bool     `json:"Online"`
			TailscaleIPs []string `json:"TailscaleIPs"`
		} `json:"Self"`
		Peer map[string]struct {
			ID           string   `json:"ID"`
			HostName     string   `json:"HostName"`
			DNSName      string   `json:"DNSName"`
			OS           string   `json:"OS"`
			Online       bool     `json:"Online"`
			TailscaleIPs []string `json:"TailscaleIPs"`
		} `json:"Peer"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return nil, fmt.Errorf("decoding tailscale status: %w", err)
	}

	if status.BackendState != "Running" {
		return []Peer{}, nil
	}

	peers := make([]Peer, 0, 1+len(status.Peer))

	peers = append(peers, Peer{
		ID:       status.Self.ID,
		Hostname: status.Self.HostName,
		DNSName:  strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), "."),
		OS:       status.Self.OS,
		Online:   status.Self.Online,
		IPs:      status.Self.TailscaleIPs,
		Self:     true,
	})

	for _, p := range status.Peer {
		peers = append(peers, Peer{
			ID:       p.ID,
			Hostname: p.HostName,
			DNSName:  strings.TrimSuffix(strings.TrimSpace(p.DNSName), "."),
			OS:       p.OS,
			Online:   p.Online,
			IPs:      p.TailscaleIPs,
			Self:     false,
		})
	}

	return peers, nil
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
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("tailscale %s: %w", strings.Join(args, " "), ctx.Err())
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return "", fmt.Errorf("tailscale %s: %s", strings.Join(args, " "), stderr)
			}
		}

		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
