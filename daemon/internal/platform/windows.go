// Package platform provides OS-specific functionality for discovering
// open terminal windows. Each platform has its own implementation file
// (windows_linux.go, windows_darwin.go) with build tags.
package platform

// TerminalWindow represents an open terminal window on the desktop.
type TerminalWindow struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	App     string `json:"app"`     // e.g., "Ghostty", "Alacritty", "Terminal"
	Focused bool   `json:"focused"`
}

// WindowDiscoverer finds open terminal windows on the desktop.
type WindowDiscoverer interface {
	// ListWindows returns all open terminal emulator windows.
	ListWindows() ([]TerminalWindow, error)
}
