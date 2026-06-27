package supacode

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListRepos(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true,"data":[{"id":"repo-1"},{"id":"repo-2"}]}`)
	})

	c := NewClient(sock)
	repos, err := c.ListRepos()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].ID != "repo-1" {
		t.Errorf("expected repo-1, got %s", repos[0].ID)
	}
	if repos[1].ID != "repo-2" {
		t.Errorf("expected repo-2, got %s", repos[1].ID)
	}
}

func TestListWorktrees(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true,"data":[{"id":"wt-1","focused":"1"},{"id":"wt-2","focused":"0"}]}`)
	})

	c := NewClient(sock)
	wts, err := c.ListWorktrees()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(wts))
	}
	if wts[0].Focused != FocusedStateActive {
		t.Errorf("expected wt-1 focused=Active, got %q", wts[0].Focused)
	}
	if wts[1].Focused != FocusedStateInactive {
		t.Errorf("expected wt-2 focused=Inactive, got %q", wts[1].Focused)
	}
}

func TestListTabs(t *testing.T) {
	t.Parallel()
	var received queryRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true,"data":[{"id":"tab-1","focused":"1"}]}`)
	})

	c := NewClient(sock)
	tabs, err := c.ListTabs("wt-1")
	if err != nil {
		t.Fatal(err)
	}
	if received.WorktreeID != "wt-1" {
		t.Errorf("expected worktreeID=wt-1, got %s", received.WorktreeID)
	}
	if len(tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(tabs))
	}
	if tabs[0].ID != "tab-1" {
		t.Errorf("expected tab-1, got %s", tabs[0].ID)
	}
}

func TestListSurfaces(t *testing.T) {
	t.Parallel()
	var received queryRequest
	sock := mockServer(t, func(req []byte) []byte {
		_ = json.Unmarshal(req, &received)
		return []byte(`{"ok":true,"data":[{"id":"surf-1","focused":"0"}]}`)
	})

	c := NewClient(sock)
	surfaces, err := c.ListSurfaces("wt-1", "tab-1")
	if err != nil {
		t.Fatal(err)
	}
	if received.WorktreeID != "wt-1" {
		t.Errorf("expected worktreeID=wt-1, got %s", received.WorktreeID)
	}
	if received.TabID != "tab-1" {
		t.Errorf("expected tabID=tab-1, got %s", received.TabID)
	}
	if len(surfaces) != 1 {
		t.Fatalf("expected 1 surface, got %d", len(surfaces))
	}
	if surfaces[0].Focused != FocusedStateInactive {
		t.Errorf("expected surface focused=Inactive, got %q", surfaces[0].Focused)
	}
}

func TestListScripts(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true,"data":[{"id":"s1","kind":"task","name":"build","displayName":"Build","running":"1"}]}`)
	})

	c := NewClient(sock)
	scripts, err := c.ListScripts("wt-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(scripts))
	}
	s := scripts[0]
	if s.ID != "s1" {
		t.Errorf("expected id=s1, got %s", s.ID)
	}
	if s.Kind != "task" {
		t.Errorf("expected kind=task, got %s", s.Kind)
	}
	if s.Name != "build" {
		t.Errorf("expected name=build, got %s", s.Name)
	}
	if s.DisplayName != "Build" {
		t.Errorf("expected displayName=Build, got %s", s.DisplayName)
	}
	if s.Running != RunningStateActive {
		t.Errorf("expected running=Active, got %q", s.Running)
	}
}

func TestQuery_EmptyIDs(t *testing.T) {
	t.Parallel()
	c := NewClient("/nonexistent")

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{"ListTabs", func() error { _, err := c.ListTabs(""); return err }, "worktreeID is required"},
		{"ListSurfaces/worktree", func() error { _, err := c.ListSurfaces("", "t"); return err }, "worktreeID is required"},
		{"ListSurfaces/tab", func() error { _, err := c.ListSurfaces("w", ""); return err }, "tabID is required"},
		{"ListScripts", func() error { _, err := c.ListScripts(""); return err }, "worktreeID is required"},
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

func TestListRepos_Empty(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":true,"data":[]}`)
	})

	c := NewClient(sock)
	repos, err := c.ListRepos()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
}

func TestListRepos_Error(t *testing.T) {
	t.Parallel()
	sock := mockServer(t, func(req []byte) []byte {
		return []byte(`{"ok":false,"error":"internal"}`)
	})

	c := NewClient(sock)
	_, err := c.ListRepos()
	if err == nil {
		t.Error("expected error")
	}
}
