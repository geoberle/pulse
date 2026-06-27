package supacode

import (
	"fmt"
	"net/url"
)

// SplitDirection represents a split orientation.
type SplitDirection string

const (
	SplitHorizontal SplitDirection = "h"
	SplitVertical   SplitDirection = "v"
)

// FocusWorktree brings a worktree to focus.
func (c *Client) FocusWorktree(worktreeID string) error {
	if len(worktreeID) == 0 {
		return fmt.Errorf("worktreeID is required")
	}
	return doCommand(c.socketPath, fmt.Sprintf("supacode://worktree/%s", worktreeID))
}

// NewTab creates a new tab in a worktree, optionally running a command.
func (c *Client) NewTab(worktreeID string, input string) error {
	if len(worktreeID) == 0 {
		return fmt.Errorf("worktreeID is required")
	}
	u := fmt.Sprintf("supacode://worktree/%s/tab/new", worktreeID)
	if len(input) > 0 {
		u += "?input=" + url.QueryEscape(input)
	}
	return doCommand(c.socketPath, u)
}

// CloseTab destroys a tab.
func (c *Client) CloseTab(tabID string) error {
	if len(tabID) == 0 {
		return fmt.Errorf("tabID is required")
	}
	return doCommand(c.socketPath, fmt.Sprintf("supacode://tab/%s/destroy", tabID))
}

// FocusTab brings a tab to focus.
func (c *Client) FocusTab(tabID string) error {
	if len(tabID) == 0 {
		return fmt.Errorf("tabID is required")
	}
	return doCommand(c.socketPath, fmt.Sprintf("supacode://tab/%s", tabID))
}

// SplitSurface creates a horizontal or vertical split in a tab.
func (c *Client) SplitSurface(worktreeID, tabID, surfaceID string, direction SplitDirection, input string) error {
	if len(worktreeID) == 0 {
		return fmt.Errorf("worktreeID is required")
	}
	if len(tabID) == 0 {
		return fmt.Errorf("tabID is required")
	}
	if len(surfaceID) == 0 {
		return fmt.Errorf("surfaceID is required")
	}
	if direction != SplitHorizontal && direction != SplitVertical {
		return fmt.Errorf("direction must be %q or %q, got %q", SplitHorizontal, SplitVertical, direction)
	}
	u := fmt.Sprintf("supacode://worktree/%s/tab/%s/surface/%s/split?direction=%s",
		worktreeID, tabID, surfaceID, url.QueryEscape(string(direction)))
	if len(input) > 0 {
		u += "&input=" + url.QueryEscape(input)
	}
	return doCommand(c.socketPath, u)
}

// FocusSurface brings a surface to focus, optionally sending input.
func (c *Client) FocusSurface(surfaceID string, input string) error {
	if len(surfaceID) == 0 {
		return fmt.Errorf("surfaceID is required")
	}
	u := fmt.Sprintf("supacode://surface/%s", surfaceID)
	if len(input) > 0 {
		u += "?input=" + url.QueryEscape(input)
	}
	return doCommand(c.socketPath, u)
}

// CloseSurface destroys a surface.
func (c *Client) CloseSurface(surfaceID string) error {
	if len(surfaceID) == 0 {
		return fmt.Errorf("surfaceID is required")
	}
	return doCommand(c.socketPath, fmt.Sprintf("supacode://surface/%s/destroy", surfaceID))
}
