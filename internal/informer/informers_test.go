package informer

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/api"
	"github.com/geoberle/pulse/internal/storage"
)

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.New(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func awaitEvent(t *testing.T, ch <-chan struct{}, desc string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", desc)
	}
}

func TestPulseInformers_WorktreeAddUpdateDelete(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := newTestStore(t)
	informers := NewPulseInformers(s, 1*time.Second)

	wtInformer, wtLister := informers.Worktrees()

	addCh := make(chan struct{}, 10)
	updateCh := make(chan struct{}, 10)
	deleteCh := make(chan struct{}, 10)

	if _, err := wtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { addCh <- struct{}{} },
		UpdateFunc: func(_, _ interface{}) { updateCh <- struct{}{} },
		DeleteFunc: func(_ interface{}) { deleteCh <- struct{}{} },
	}); err != nil {
		t.Fatal(err)
	}

	go informers.RunWithContext(ctx)

	if !cache.WaitForCacheSync(ctx.Done(), wtInformer.HasSynced) {
		t.Fatal("informer did not sync")
	}

	_, err := s.Worktrees().Create(ctx, &api.Worktree{
		ObjectMeta: api.ObjectMeta{Name: "/tmp/test"},
		Repo:       "org/repo",
		Branch:     "main",
	})
	if err != nil {
		t.Fatal(err)
	}

	awaitEvent(t, addCh, "add event")

	items, err := wtLister.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item in lister, got %d", len(items))
	}

	wt, found, err := wtLister.Get("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected to find worktree by key")
	}
	if wt.Repo != "org/repo" {
		t.Errorf("expected repo org/repo, got %s", wt.Repo)
	}

	got, err := s.Worktrees().Get(ctx, "/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	got.Branch = "feature"
	if _, err := s.Worktrees().Update(ctx, got); err != nil {
		t.Fatal(err)
	}

	awaitEvent(t, updateCh, "update event")

	if err := s.Worktrees().Delete(ctx, "/tmp/test"); err != nil {
		t.Fatal(err)
	}

	awaitEvent(t, deleteCh, "delete event")
}

func TestPulseInformers_WorktreeIndex(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := newTestStore(t)
	informers := NewPulseInformers(s, 1*time.Second)

	_, wtLister := informers.Worktrees()

	for _, wt := range []*api.Worktree{
		{ObjectMeta: api.ObjectMeta{Name: "/tmp/a"}, Repo: "org/repo-a", Branch: "main"},
		{ObjectMeta: api.ObjectMeta{Name: "/tmp/b"}, Repo: "org/repo-b", Branch: "main"},
		{ObjectMeta: api.ObjectMeta{Name: "/tmp/c"}, Repo: "org/repo-a", Branch: "feature"},
	} {
		if _, err := s.Worktrees().Create(ctx, wt); err != nil {
			t.Fatal(err)
		}
	}

	go informers.RunWithContext(ctx)

	wtInformer, _ := informers.Worktrees()
	if !cache.WaitForCacheSync(ctx.Done(), wtInformer.HasSynced) {
		t.Fatal("informer did not sync")
	}

	repoA, err := wtLister.ByIndex(ByRepo, "org/repo-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(repoA) != 2 {
		t.Errorf("expected 2 items for repo-a, got %d", len(repoA))
	}

	repoB, err := wtLister.ByIndex(ByRepo, "org/repo-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(repoB) != 1 {
		t.Errorf("expected 1 item for repo-b, got %d", len(repoB))
	}
}
