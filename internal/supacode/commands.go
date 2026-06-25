package supacode

import (
	"fmt"
	"net/url"
)

// FocusWorktree brings a worktree to focus.
func (c *Client) FocusWorktree(worktreeID string) error {
	return doCommand(c.socketPath, fmt.Sprintf("supacode://worktree/%s", worktreeID))
}

// NewTab creates a new tab in a worktree, optionally running a command.
func (c *Client) NewTab(worktreeID string, input string) error {
	u := fmt.Sprintf("supacode://worktree/%s/tab/new", worktreeID)
	if len(input) > 0 {
		u += "?input=" + url.QueryEscape(input)
	}
	return doCommand(c.socketPath, u)
}

// CloseTab destroys a tab.
func (c *Client) CloseTab(tabID string) error {
	return doCommand(c.socketPath, fmt.Sprintf("supacode://tab/%s/destroy", tabID))
}

// FocusTab brings a tab to focus.
func (c *Client) FocusTab(tabID string) error {
	return doCommand(c.socketPath, fmt.Sprintf("supacode://tab/%s", tabID))
}

// SplitSurface creates a horizontal or vertical split in a tab.
// direction must be "h" or "v".
func (c *Client) SplitSurface(worktreeID, tabID, surfaceID, direction, input string) error {
	if direction != "h" && direction != "v" {
		return fmt.Errorf("direction must be %q or %q, got %q", "h", "v", direction)
	}
	u := fmt.Sprintf("supacode://worktree/%s/tab/%s/surface/%s/split?direction=%s",
		worktreeID, tabID, surfaceID, url.QueryEscape(direction))
	if len(input) > 0 {
		u += "&input=" + url.QueryEscape(input)
	}
	return doCommand(c.socketPath, u)
}

// FocusSurface brings a surface to focus, optionally sending input.
func (c *Client) FocusSurface(surfaceID string, input string) error {
	u := fmt.Sprintf("supacode://surface/%s", surfaceID)
	if len(input) > 0 {
		u += "?input=" + url.QueryEscape(input)
	}
	return doCommand(c.socketPath, u)
}

// CloseSurface destroys a surface.
func (c *Client) CloseSurface(surfaceID string) error {
	return doCommand(c.socketPath, fmt.Sprintf("supacode://surface/%s/destroy", surfaceID))
}
