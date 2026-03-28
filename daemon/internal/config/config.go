// Package config manages iTTY daemon configuration.
// Configuration is loaded from ~/.config/itty/config.toml
// with sensible defaults for all values.
package config

import (
	"os"
	"path/filepath"
)

// Config holds all daemon configuration.
type Config struct {
	// Server settings
	ListenAddr string `toml:"listen_addr"` // Address to bind HTTP server (default: ":8080")

	// tmux settings
	TmuxPath   string `toml:"tmux_path"`   // Path to tmux binary (default: "tmux")
	AutoWrap   bool   `toml:"auto_wrap"`   // Auto-configure shell for tmux wrapping (default: true)

	// Tailscale settings
	TailscaleServe bool `toml:"tailscale_serve"` // Auto-configure tailscale serve (default: true)

	// Notification settings
	APNsKeyPath  string `toml:"apns_key_path"`  // Path to APNs .p8 key file
	APNsKeyID    string `toml:"apns_key_id"`    // APNs key ID
	APNsTeamID   string `toml:"apns_team_id"`   // Apple Developer Team ID
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:     ":8080",
		TmuxPath:       "tmux",
		AutoWrap:       true,
		TailscaleServe: true,
	}
}

// ConfigDir returns the path to the iTTY config directory.
// Creates it if it doesn't exist.
func ConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configHome, "itty")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ConfigPath returns the full path to the config file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}
