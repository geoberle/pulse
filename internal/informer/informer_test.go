package informer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"

	"github.com/geoberle/pulse/internal/api"
	"github.com/geoberle/pulse/internal/storage"
)

func testLog() logr.Logger {
	return funcr.New(func(_, _ string) {}, funcr.Options{})
}

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.New(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestInformer_AddUpdateDelete(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := newTestStore(t)
	inf := New[*api.Worktree](testLog(), s.Worktrees().List, 50*time.Millisecond)

	var mu sync.Mutex
	var adds, updates, deletes []*api.Worktree

	inf.AddEventHandler(ResourceEventHandlerFuncs[*api.Worktree]{
		AddFunc: func(obj *api.Worktree) {
			mu.Lock()
			adds = append(adds, obj)
			mu.Unlock()
		},
		UpdateFunc: func(_, newObj *api.Worktree) {
			mu.Lock()
			updates = append(updates, newObj)
			mu.Unlock()
		},
		DeleteFunc: func(obj *api.Worktree) {
			mu.Lock()
			deletes = append(deletes, obj)
			mu.Unlock()
		},
	})

	go inf.Run(ctx)

	// wait for initial sync
	for i := range 100 {
		if inf.HasSynced() {
			break
		}
		if i == 99 {
			t.Fatal("informer did not sync")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// create a worktree — should trigger OnAdd on next poll
	_, err := s.Worktrees().Create(ctx, &api.Worktree{
		Path: "/tmp/test", Repo: "org/repo", Branch: "main",
		LastSeen: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	if len(adds) != 1 {
		t.Errorf("expected 1 add, got %d", len(adds))
	}
	mu.Unlock()

	// update — should trigger OnUpdate
	wt, err := s.Worktrees().Get(ctx, "/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	wt.Branch = "feature"
	if _, err := s.Worktrees().Update(ctx, wt); err != nil {
		t.Fatal(err)
	}

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	if len(updates) != 1 {
		t.Errorf("expected 1 update, got %d", len(updates))
	}
	mu.Unlock()

	// delete — should trigger OnDelete
	if err := s.Worktrees().Delete(ctx, "/tmp/test"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	if len(deletes) != 1 {
		t.Errorf("expected 1 delete, got %d", len(deletes))
	}
	mu.Unlock()
}

func TestInformer_ListerAndIndex(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := newTestStore(t)
	inf := New[*api.Worktree](testLog(), s.Worktrees().List, 50*time.Millisecond)

	inf.AddIndexer("repo", func(obj *api.Worktree) []string {
		return []string{obj.Repo}
	})

	now := time.Now()
	if _, err := s.Worktrees().Create(ctx, &api.Worktree{
		Path: "/tmp/a", Repo: "org/repo-a", Branch: "main", LastSeen: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Worktrees().Create(ctx, &api.Worktree{
		Path: "/tmp/b", Repo: "org/repo-b", Branch: "main", LastSeen: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Worktrees().Create(ctx, &api.Worktree{
		Path: "/tmp/c", Repo: "org/repo-a", Branch: "feature", LastSeen: now,
	}); err != nil {
		t.Fatal(err)
	}

	go inf.Run(ctx)
	for i := range 100 {
		if inf.HasSynced() {
			break
		}
		if i == 99 {
			t.Fatal("informer did not sync")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// lister
	lister := inf.Lister()
	all := lister.List()
	if len(all) != 3 {
		t.Errorf("expected 3 items, got %d", len(all))
	}

	// get by key
	item, ok := lister.Get("/tmp/a")
	if !ok {
		t.Fatal("expected to find /tmp/a")
	}
	if item.Repo != "org/repo-a" {
		t.Errorf("expected repo org/repo-a, got %s", item.Repo)
	}

	// get nonexistent
	_, ok = lister.Get("/tmp/nonexistent")
	if ok {
		t.Error("expected not found for nonexistent key")
	}

	// index
	repoA := lister.ByIndex("repo", "org/repo-a")
	if len(repoA) != 2 {
		t.Errorf("expected 2 items for repo-a, got %d", len(repoA))
	}

	repoB := lister.ByIndex("repo", "org/repo-b")
	if len(repoB) != 1 {
		t.Errorf("expected 1 item for repo-b, got %d", len(repoB))
	}

	// nonexistent index
	nope := lister.ByIndex("nonexistent", "value")
	if len(nope) != 0 {
		t.Errorf("expected 0 items for nonexistent index, got %d", len(nope))
	}
}
