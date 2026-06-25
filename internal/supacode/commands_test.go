package supacode

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFocusWorktree(t *testing.T) {
	t.Parallel()
	var received commandRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true}`)
	})

	c := NewClient(sock)
	if err := c.FocusWorktree("wt-1"); err != nil {
		t.Fatal(err)
	}
	if received.Deeplink != "supacode://worktree/wt-1" {
		t.Errorf("expected supacode://worktree/wt-1, got %s", received.Deeplink)
	}
}

func TestNewTab(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantURL string
	}{
		{
			name:    "no input",
			input:   "",
			wantURL: "supacode://worktree/wt-1/tab/new",
		},
		{
			name:    "with input",
			input:   "echo hello",
			wantURL: "supacode://worktree/wt-1/tab/new?input=echo+hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var received commandRequest
			sock := mockServer(t, func(req []byte) []byte {
				_ = json.Unmarshal(req, &received)
				return []byte(`{"ok":true}`)
			})

			c := NewClient(sock)
			if err := c.NewTab("wt-1", tt.input); err != nil {
				t.Fatal(err)
			}
			if received.Deeplink != tt.wantURL {
				t.Errorf("expected %s, got %s", tt.wantURL, received.Deeplink)
			}
		})
	}
}

func TestCloseTab(t *testing.T) {
	t.Parallel()
	var received commandRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true}`)
	})

	c := NewClient(sock)
	if err := c.CloseTab("tab-1"); err != nil {
		t.Fatal(err)
	}
	if received.Deeplink != "supacode://tab/tab-1/destroy" {
		t.Errorf("expected supacode://tab/tab-1/destroy, got %s", received.Deeplink)
	}
}

func TestFocusTab(t *testing.T) {
	t.Parallel()
	var received commandRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true}`)
	})

	c := NewClient(sock)
	if err := c.FocusTab("tab-1"); err != nil {
		t.Fatal(err)
	}
	if received.Deeplink != "supacode://tab/tab-1" {
		t.Errorf("expected supacode://tab/tab-1, got %s", received.Deeplink)
	}
}

func TestSplitSurface(t *testing.T) {
	t.Parallel()
	var received commandRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true}`)
	})

	c := NewClient(sock)
	if err := c.SplitSurface("wt-1", "tab-1", "surf-1", "h", "ls"); err != nil {
		t.Fatal(err)
	}
	want := "supacode://worktree/wt-1/tab/tab-1/surface/surf-1/split?direction=h&input=ls"
	if received.Deeplink != want {
		t.Errorf("expected %s, got %s", want, received.Deeplink)
	}
}

func TestSplitSurface_NoInput(t *testing.T) {
	t.Parallel()
	var received commandRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true}`)
	})

	c := NewClient(sock)
	if err := c.SplitSurface("wt-1", "tab-1", "surf-1", "v", ""); err != nil {
		t.Fatal(err)
	}
	want := "supacode://worktree/wt-1/tab/tab-1/surface/surf-1/split?direction=v"
	if received.Deeplink != want {
		t.Errorf("expected %s, got %s", want, received.Deeplink)
	}
}

func TestFocusSurface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantURL string
	}{
		{
			name:    "no input",
			input:   "",
			wantURL: "supacode://surface/surf-1",
		},
		{
			name:    "with input",
			input:   "pwd",
			wantURL: "supacode://surface/surf-1?input=pwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var received commandRequest
			sock := mockServer(t, func(req []byte) []byte {
				_ = json.Unmarshal(req, &received)
				return []byte(`{"ok":true}`)
			})

			c := NewClient(sock)
			if err := c.FocusSurface("surf-1", tt.input); err != nil {
				t.Fatal(err)
			}
			if received.Deeplink != tt.wantURL {
				t.Errorf("expected %s, got %s", tt.wantURL, received.Deeplink)
			}
		})
	}
}

func TestCloseSurface(t *testing.T) {
	t.Parallel()
	var received commandRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true}`)
	})

	c := NewClient(sock)
	if err := c.CloseSurface("surf-1"); err != nil {
		t.Fatal(err)
	}
	if received.Deeplink != "supacode://surface/surf-1/destroy" {
		t.Errorf("expected supacode://surface/surf-1/destroy, got %s", received.Deeplink)
	}
}

func TestSplitSurface_InvalidDirection(t *testing.T) {
	t.Parallel()
	c := NewClient("/nonexistent")
	err := c.SplitSurface("wt-1", "tab-1", "surf-1", "x", "")
	if err == nil {
		t.Error("expected error for invalid direction")
	}
	if !strings.Contains(err.Error(), "direction must be") {
		t.Errorf("expected direction validation error, got: %v", err)
	}
}

func TestCommand_EmptyIDs(t *testing.T) {
	t.Parallel()
	c := NewClient("/nonexistent")

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{"FocusWorktree", func() error { return c.FocusWorktree("") }, "worktreeID is required"},
		{"NewTab", func() error { return c.NewTab("", "ls") }, "worktreeID is required"},
		{"CloseTab", func() error { return c.CloseTab("") }, "tabID is required"},
		{"FocusTab", func() error { return c.FocusTab("") }, "tabID is required"},
		{"SplitSurface/worktree", func() error { return c.SplitSurface("", "t", "s", "h", "") }, "worktreeID is required"},
		{"SplitSurface/tab", func() error { return c.SplitSurface("w", "", "s", "h", "") }, "tabID is required"},
		{"SplitSurface/surface", func() error { return c.SplitSurface("w", "t", "", "h", "") }, "surfaceID is required"},
		{"FocusSurface", func() error { return c.FocusSurface("", "") }, "surfaceID is required"},
		{"CloseSurface", func() error { return c.CloseSurface("") }, "surfaceID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.fn()
			if err == nil {
				t.Error("expected error for empty ID")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected %q in error, got: %v", tt.want, err)
			}
		})
	}
}

func TestCommand_ServerError(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":false,"error":"bad request"}`)
	})

	c := NewClient(sock)
	err := c.FocusWorktree("wt-1")
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Errorf("expected 'bad request' in error, got: %v", err)
	}
}
