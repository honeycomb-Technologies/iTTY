package tmux

import (
	"context"
	"fmt"
	"strings"
)

// CapturePaneContent captures the visible content of a pane.
// Returns the text currently displayed in the pane as a string.
func (c *Client) CapturePaneContent(ctx context.Context, sessionName string, windowIndex int, paneIndex int) (string, error) {
	target := fmt.Sprintf("%s:%d.%d", sessionName, windowIndex, paneIndex)
	return c.run(ctx, "capture-pane", "-t", target, "-p")
}

// CapturePaneByID captures pane content using the tmux pane ID (e.g., "%42").
func (c *Client) CapturePaneByID(ctx context.Context, paneID string) (string, error) {
	return c.run(ctx, "capture-pane", "-t", paneID, "-p")
}

// CaptureSessionDefaultPane captures the active pane of the active window
// in the given session.
func (c *Client) CaptureSessionDefaultPane(ctx context.Context, sessionName string) (string, error) {
	content, err := c.run(ctx, "capture-pane", "-t", sessionName, "-p")
	if err != nil {
		if strings.Contains(err.Error(), "can't find session") ||
			strings.Contains(err.Error(), "can't find pane") {
			return "", fmt.Errorf("%w: %s", ErrSessionNotFound, sessionName)
		}
		return "", err
	}

	return content, nil
}
