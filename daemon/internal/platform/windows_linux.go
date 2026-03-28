//go:build linux

package platform

import (
	"os/exec"
	"strings"
)

// LinuxWindowDiscoverer finds terminal windows using wmctrl or xdotool.
type LinuxWindowDiscoverer struct{}

// NewWindowDiscoverer returns the platform-specific window discoverer.
func NewWindowDiscoverer() WindowDiscoverer {
	return &LinuxWindowDiscoverer{}
}

// ListWindows lists open terminal windows on Linux.
// Uses wmctrl if available, falls back to empty list.
func (d *LinuxWindowDiscoverer) ListWindows() ([]TerminalWindow, error) {
	// Check for wmctrl
	wmctrlPath, err := exec.LookPath("wmctrl")
	if err != nil {
		// wmctrl not installed — return empty list, not an error.
		// Window discovery is a nice-to-have, not a requirement.
		return []TerminalWindow{}, nil
	}

	out, err := exec.Command(wmctrlPath, "-l").Output()
	if err != nil {
		return []TerminalWindow{}, nil
	}

	// Known terminal emulator WM classes/names
	terminalApps := map[string]bool{
		"ghostty":        true,
		"alacritty":      true,
		"kitty":          true,
		"wezterm":        true,
		"foot":           true,
		"gnome-terminal": true,
		"konsole":        true,
		"xterm":          true,
		"tilix":          true,
	}

	var windows []TerminalWindow
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// wmctrl -l format: <id> <desktop> <host> <title>
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		title := strings.Join(fields[3:], " ")
		titleLower := strings.ToLower(title)

		// Check if this looks like a terminal window
		for app := range terminalApps {
			if strings.Contains(titleLower, app) {
				windows = append(windows, TerminalWindow{
					ID:    fields[0],
					Title: title,
					App:   app,
				})
				break
			}
		}
	}

	if windows == nil {
		windows = []TerminalWindow{}
	}
	return windows, nil
}
