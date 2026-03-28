// Package shell handles detection and auto-configuration of the user's shell
// for automatic tmux session wrapping. When enabled, every new terminal window
// automatically runs inside a tmux session — making it accessible from iTTY.
package shell

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ShellType represents a supported shell.
type ShellType string

const (
	Bash ShellType = "bash"
	Zsh  ShellType = "zsh"
	Fish ShellType = "fish"
)

// ShellInfo contains detected shell information.
type ShellInfo struct {
	Type   ShellType
	Path   string // absolute path to shell binary
	RCFile string // path to the rc file we'd modify
}

// Detect identifies the user's default shell and its rc file.
func Detect() (*ShellInfo, error) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		// Fallback: read from /etc/passwd
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("cannot detect shell: %w", err)
		}
		// user.Current() doesn't give shell on all systems,
		// fall back to bash
		_ = u
		shellPath = "/bin/bash"
	}

	base := filepath.Base(shellPath)
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home directory: %w", err)
	}

	info := &ShellInfo{Path: shellPath}

	switch {
	case strings.Contains(base, "zsh"):
		info.Type = Zsh
		info.RCFile = filepath.Join(home, ".zshrc")
	case strings.Contains(base, "bash"):
		info.Type = Bash
		// Prefer .bashrc for interactive shells, but check .bash_profile too
		info.RCFile = filepath.Join(home, ".bashrc")
	case strings.Contains(base, "fish"):
		info.Type = Fish
		info.RCFile = filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return nil, fmt.Errorf("unsupported shell: %s", base)
	}

	return info, nil
}
