// Package config manages iTTY daemon configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all daemon configuration.
type Config struct {
	// Server settings
	ListenAddr string `toml:"listen_addr" json:"listenAddr"` // Address to bind HTTP server (default: ":3420")

	// tmux settings
	TmuxPath string `toml:"tmux_path" json:"tmuxPath"` // Path to tmux binary (default: "tmux")
	AutoWrap bool   `toml:"auto_wrap" json:"autoWrap"` // Auto-configure shell for tmux wrapping (default: true)

	// Tailscale settings
	TailscaleServe bool `toml:"tailscale_serve" json:"tailscaleServe"` // Auto-configure tailscale serve (default: true)

	// Notification settings
	APNsKeyPath    string `toml:"apns_key_path" json:"apnsKeyPath"`       // Path to APNs .p8 key file
	APNsKeyID      string `toml:"apns_key_id" json:"apnsKeyID"`           // APNs key ID
	APNsTeamID     string `toml:"apns_team_id" json:"apnsTeamID"`         // Apple Developer Team ID
	APNsProduction bool   `toml:"apns_production" json:"apnsProduction"` // Use production APNs gateway (default: false)
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:     ":3420",
		TmuxPath:       "tmux",
		AutoWrap:       true,
		TailscaleServe: true,
	}
}

// Validate checks whether the configuration is internally consistent.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	if c.ListenAddr == "" {
		return errors.New("listen address is required")
	}
	if c.TmuxPath == "" {
		return errors.New("tmux path is required")
	}
	return nil
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

// Load reads configuration from the given path and overlays it onto defaults.
// A missing file returns DefaultConfig with no error.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if _, err := toml.Decode(string(content), cfg); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating %s: %w", path, err)
	}

	return cfg, nil
}

// LoadDefault reads configuration from the default config path.
func LoadDefault() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return Load(path)
}

// Save writes configuration to the given path atomically.
func Save(path string, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir for %s: %w", path, err)
	}

	data, err := toml.Marshal(*cfg)
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmpPath, path, err)
	}

	return nil
}

// SaveDefault writes configuration to the default config path.
func SaveDefault(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return Save(path, cfg)
}
