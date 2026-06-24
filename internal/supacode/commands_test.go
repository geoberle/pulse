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
		json.Unmarshal(req, &received)
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
		name     string
		input    string
		wantURL  string
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
				json.Unmarshal(req, &received)
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
		json.Unmarshal(req, &received)
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
		json.Unmarshal(req, &received)
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
		json.Unmarshal(req, &received)
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
		json.Unmarshal(req, &received)
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
				json.Unmarshal(req, &received)
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
		json.Unmarshal(req, &received)
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
