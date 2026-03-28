//go:build darwin

package platform

import (
	"os/exec"
	"strings"
)

// DarwinWindowDiscoverer finds terminal windows using AppleScript.
type DarwinWindowDiscoverer struct{}

// NewWindowDiscoverer returns the platform-specific window discoverer.
func NewWindowDiscoverer() WindowDiscoverer {
	return &DarwinWindowDiscoverer{}
}

// ListWindows lists open terminal windows on macOS using AppleScript.
func (d *DarwinWindowDiscoverer) ListWindows() ([]TerminalWindow, error) {
	// AppleScript to list windows of known terminal apps
	script := `
		set output to ""
		set termApps to {"Ghostty", "Terminal", "iTerm2", "Alacritty", "kitty", "WezTerm"}
		repeat with appName in termApps
			try
				tell application "System Events"
					if exists process appName then
						tell process appName
							set windowCount to count of windows
							repeat with i from 1 to windowCount
								set winTitle to name of window i
								set output to output & appName & "|" & winTitle & linefeed
							end repeat
						end tell
					end if
				end tell
			end try
		end repeat
		return output
	`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return []TerminalWindow{}, nil
	}

	var windows []TerminalWindow
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		windows = append(windows, TerminalWindow{
			App:   parts[0],
			Title: parts[1],
		})
	}

	if windows == nil {
		windows = []TerminalWindow{}
	}
	return windows, nil
}
