package informer

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/workitem"
)

type staticSource struct {
	mu    sync.Mutex
	items []*workitem.WorkItem
}

func (s *staticSource) List(_ context.Context) ([]*workitem.WorkItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]*workitem.WorkItem, len(s.items))
	copy(cp, s.items)
	return cp, nil
}

type recordingHandler struct {
	mu      sync.Mutex
	added   []runtime.Object
	updated []runtime.Object
	deleted []runtime.Object
}

func (h *recordingHandler) OnAdd(obj interface{}, _ bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.added = append(h.added, obj.(runtime.Object))
}

func (h *recordingHandler) OnUpdate(_, newObj interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updated = append(h.updated, newObj.(runtime.Object))
}

func (h *recordingHandler) OnDelete(obj interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deleted = append(h.deleted, obj.(runtime.Object))
}

func (h *recordingHandler) counts() (int, int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.added), len(h.updated), len(h.deleted)
}

func (h *recordingHandler) getAdded() []runtime.Object {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make([]runtime.Object, len(h.added))
	copy(cp, h.added)
	return cp
}

func waitForSync(t *testing.T, inf cache.SharedIndexInformer, timeout time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if !cache.WaitForCacheSync(ctx.Done(), inf.HasSynced) {
		t.Fatal("informer did not sync within timeout")
	}
}

func TestNew_CreatesWorkingInformer(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	src := &staticSource{items: items}
	inf := New(src, time.Hour)
	h := &recordingHandler{}
	if _, err := inf.AddEventHandler(h); err != nil {
		t.Fatalf("AddEventHandler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go inf.Run(ctx.Done())
	waitForSync(t, inf, 5*time.Second)

	added, _, _ := h.counts()
	if added != 1 {
		t.Fatalf("expected 1 added event, got %d", added)
	}
	addedItems := h.getAdded()
	item := addedItems[0].(*workitem.WorkItem)
	if item.ID != "jira:ARO-1" {
		t.Errorf("expected ID jira:ARO-1, got %s", item.ID)
	}
}

func TestNew_FlattensTree(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Root", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
			Children: []*workitem.WorkItem{
				{
					TypeMeta:   workitem.TypeMeta{Kind: workitem.KindPR},
					ObjectMeta: workitem.ObjectMeta{ID: "pr:repo:1", Label: "PR", Status: "open"},
					Spec:       json.RawMessage(`{"repo":"test","number":1}`),
				},
			},
		},
	}
	src := &staticSource{items: items}
	inf := New(src, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go inf.Run(ctx.Done())
	waitForSync(t, inf, 5*time.Second)

	lister := NewLister(inf.GetIndexer())
	all, err := lister.List()
	if err != nil {
		t.Fatalf("lister.List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 flat items, got %d", len(all))
	}
}

func TestByParentIndex(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Root", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
			Children: []*workitem.WorkItem{
				{
					TypeMeta:   workitem.TypeMeta{Kind: workitem.KindPR},
					ObjectMeta: workitem.ObjectMeta{ID: "pr:repo:1", Label: "PR 1", Status: "open"},
					Spec:       json.RawMessage(`{"repo":"test","number":1}`),
				},
				{
					TypeMeta:   workitem.TypeMeta{Kind: workitem.KindPR},
					ObjectMeta: workitem.ObjectMeta{ID: "pr:repo:2", Label: "PR 2", Status: "open"},
					Spec:       json.RawMessage(`{"repo":"test","number":2}`),
				},
			},
		},
	}
	src := &staticSource{items: items}
	inf := New(src, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go inf.Run(ctx.Done())
	waitForSync(t, inf, 5*time.Second)

	lister := NewLister(inf.GetIndexer())

	roots, err := lister.ByIndex(ByParent, "")
	if err != nil {
		t.Fatalf("ByIndex(ByParent, ''): %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}

	children, err := lister.ByIndex(ByParent, "jira:ARO-1")
	if err != nil {
		t.Fatalf("ByIndex(ByParent, 'jira:ARO-1'): %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
}

func TestByKindIndex(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Root", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
			Children: []*workitem.WorkItem{
				{
					TypeMeta:   workitem.TypeMeta{Kind: workitem.KindPR},
					ObjectMeta: workitem.ObjectMeta{ID: "pr:repo:1", Label: "PR", Status: "open"},
					Spec:       json.RawMessage(`{"repo":"test","number":1}`),
				},
			},
		},
	}
	src := &staticSource{items: items}
	inf := New(src, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go inf.Run(ctx.Done())
	waitForSync(t, inf, 5*time.Second)

	lister := NewLister(inf.GetIndexer())

	jiras, err := lister.ByIndex(ByKind, "jira")
	if err != nil {
		t.Fatalf("ByIndex(ByKind, 'jira'): %v", err)
	}
	if len(jiras) != 1 {
		t.Fatalf("expected 1 jira item, got %d", len(jiras))
	}

	prs, err := lister.ByIndex(ByKind, "pr")
	if err != nil {
		t.Fatalf("ByIndex(ByKind, 'pr'): %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("expected 1 pr item, got %d", len(prs))
	}
}

func TestLister_Get(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	src := &staticSource{items: items}
	inf := New(src, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go inf.Run(ctx.Done())
	waitForSync(t, inf, 5*time.Second)

	lister := NewLister(inf.GetIndexer())

	item, exists, err := lister.Get("jira:ARO-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !exists {
		t.Fatal("expected item to exist")
	}
	if item.Label != "Test" {
		t.Errorf("expected label Test, got %s", item.Label)
	}

	_, exists, err = lister.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get nonexistent: %v", err)
	}
	if exists {
		t.Error("expected nonexistent item to not exist")
	}
}

func TestExpiringWatcher_Expires(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	w := newExpiringWatcher(ctx, 50*time.Millisecond)
	defer w.Stop()

	select {
	case event := <-w.ResultChan():
		if event.Type != "ERROR" {
			t.Errorf("expected ERROR event, got %s", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not expire within timeout")
	}
}

func TestExpiringWatcher_Stop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	w := newExpiringWatcher(ctx, time.Hour)
	w.Stop()

	select {
	case _, ok := <-w.ResultChan():
		if ok {
			t.Fatal("expected channel to close after Stop")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ResultChan did not close after Stop")
	}
}
