package informer

import (
	"encoding/json"
	"testing"

	"github.com/geoberle/pulse/internal/workitem"
)

type recordingHandler struct {
	events []Event
}

func (h *recordingHandler) OnEvent(e Event) {
	h.events = append(h.events, e)
}

func TestSync_DispatchesToHandlers(t *testing.T) {
	t.Parallel()
	inf := New()
	h1 := &recordingHandler{}
	h2 := &recordingHandler{}
	inf.RegisterHandler(h1)
	inf.RegisterHandler(h2)

	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}

	inf.Sync(items)

	if len(h1.events) != 1 {
		t.Fatalf("handler 1: expected 1 event, got %d", len(h1.events))
	}
	if len(h2.events) != 1 {
		t.Fatalf("handler 2: expected 1 event, got %d", len(h2.events))
	}
	if h1.events[0].Type != EventAdded {
		t.Errorf("expected Added, got %s", h1.events[0].Type)
	}
}

func TestSync_SecondCallDiffs(t *testing.T) {
	t.Parallel()
	inf := New()
	h := &recordingHandler{}
	inf.RegisterHandler(h)

	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	inf.Sync(items)

	if len(h.events) != 1 {
		t.Fatalf("first sync: expected 1 event, got %d", len(h.events))
	}

	h.events = nil
	inf.Sync(items)

	if len(h.events) != 0 {
		t.Fatalf("second sync (no change): expected 0 events, got %d", len(h.events))
	}

	updated := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "Done"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	inf.Sync(updated)

	if len(h.events) != 1 {
		t.Fatalf("third sync (status change): expected 1 event, got %d", len(h.events))
	}
	if h.events[0].Type != EventUpdated {
		t.Errorf("expected Updated, got %s", h.events[0].Type)
	}
}

func TestSync_CacheUpdated(t *testing.T) {
	t.Parallel()
	inf := New()

	if len(inf.Cache()) != 0 {
		t.Fatalf("expected empty cache, got %d items", len(inf.Cache()))
	}

	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	inf.Sync(items)

	if len(inf.Cache()) != 1 {
		t.Fatalf("expected 1 cached item, got %d", len(inf.Cache()))
	}
	if inf.Cache()[0].ID != "jira:ARO-1" {
		t.Errorf("expected cached ID jira:ARO-1, got %s", inf.Cache()[0].ID)
	}
}

func TestSync_DeleteEvent(t *testing.T) {
	t.Parallel()
	inf := New()
	h := &recordingHandler{}
	inf.RegisterHandler(h)

	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	inf.Sync(items)

	h.events = nil
	inf.Sync(nil)

	if len(h.events) != 1 {
		t.Fatalf("expected 1 delete event, got %d", len(h.events))
	}
	if h.events[0].Type != EventDeleted {
		t.Errorf("expected Deleted, got %s", h.events[0].Type)
	}
	if h.events[0].Old.ID != "jira:ARO-1" {
		t.Errorf("expected old ID jira:ARO-1, got %s", h.events[0].Old.ID)
	}
}
