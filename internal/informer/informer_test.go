package informer

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/workitem"
)

type recordingHandler struct {
	mu     sync.Mutex
	events []Event
}

func (h *recordingHandler) OnEvent(e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, e)
}

func (h *recordingHandler) getEvents() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make([]Event, len(h.events))
	copy(cp, h.events)
	return cp
}

func (h *recordingHandler) resetEvents() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = nil
}

type staticSource struct {
	mu    sync.Mutex
	items []*workitem.WorkItem
}

func (s *staticSource) List(_ context.Context) ([]*workitem.WorkItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.items, nil
}

func (s *staticSource) setItems(items []*workitem.WorkItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = items
}

func TestEventType_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		et      EventType
		wantErr bool
	}{
		{name: "Added", et: EventAdded},
		{name: "Updated", et: EventUpdated},
		{name: "Deleted", et: EventDeleted},
		{name: "unknown", et: EventType("bogus"), wantErr: true},
		{name: "empty", et: EventType(""), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.et.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPoll_DispatchesAddedEvents(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	src := &staticSource{items: items}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)
	h := &recordingHandler{}
	inf.RegisterHandler(h)

	inf.poll(context.Background())

	events := h.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventAdded {
		t.Errorf("expected Added, got %s", events[0].Type)
	}
}

func TestPoll_SetsSynced(t *testing.T) {
	t.Parallel()
	src := &staticSource{items: nil}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)

	if inf.HasSynced() {
		t.Fatal("expected HasSynced=false before poll")
	}

	inf.poll(context.Background())

	if !inf.HasSynced() {
		t.Fatal("expected HasSynced=true after poll")
	}
}

func TestPoll_NoEventsOnUnchangedData(t *testing.T) {
	t.Parallel()
	item := &workitem.WorkItem{
		TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
		ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
		Spec:       json.RawMessage(`{"key":"ARO-1"}`),
	}
	src := &staticSource{items: []*workitem.WorkItem{item}}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)
	h := &recordingHandler{}
	inf.RegisterHandler(h)

	inf.poll(context.Background())
	h.resetEvents()
	inf.poll(context.Background())

	events := h.getEvents()
	if len(events) != 0 {
		t.Fatalf("expected 0 events on re-poll of unchanged data, got %d", len(events))
	}
}

func TestRelist_DeliversAllItemsAsUpdated(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-2", Label: "Test2", Status: "Done"},
			Spec:       json.RawMessage(`{"key":"ARO-2"}`),
		},
	}
	src := &staticSource{items: items}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)
	h := &recordingHandler{}
	inf.RegisterHandler(h)

	inf.poll(context.Background())
	h.resetEvents()
	inf.relist()

	events := h.getEvents()
	updatedCount := 0
	for _, e := range events {
		if e.Type == EventUpdated {
			updatedCount++
		}
	}
	if updatedCount != 2 {
		t.Errorf("expected 2 Updated events from relist, got %d", updatedCount)
	}
}

func TestPoll_DispatchesToMultipleHandlers(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	src := &staticSource{items: items}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)
	h1 := &recordingHandler{}
	h2 := &recordingHandler{}
	inf.RegisterHandler(h1)
	inf.RegisterHandler(h2)

	inf.poll(context.Background())

	if len(h1.getEvents()) == 0 {
		t.Error("handler 1 received no events")
	}
	if len(h2.getEvents()) == 0 {
		t.Error("handler 2 received no events")
	}
}

func TestPoll_DeleteEvent(t *testing.T) {
	t.Parallel()
	item := &workitem.WorkItem{
		TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
		ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
		Spec:       json.RawMessage(`{"key":"ARO-1"}`),
	}
	src := &staticSource{items: []*workitem.WorkItem{item}}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)
	h := &recordingHandler{}
	inf.RegisterHandler(h)

	inf.poll(context.Background())
	h.resetEvents()

	src.setItems(nil)
	inf.poll(context.Background())

	events := h.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 delete event, got %d", len(events))
	}
	if events[0].Type != EventDeleted {
		t.Errorf("expected Deleted, got %s", events[0].Type)
	}
	if events[0].Old.ID != "jira:ARO-1" {
		t.Errorf("expected old ID jira:ARO-1, got %s", events[0].Old.ID)
	}
}

func TestCache_ReturnsCurrentState(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	src := &staticSource{items: items}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)

	if len(inf.Cache()) != 0 {
		t.Fatalf("expected empty cache before poll, got %d", len(inf.Cache()))
	}

	inf.poll(context.Background())

	if len(inf.Cache()) != 1 {
		t.Fatalf("expected 1 cached item, got %d", len(inf.Cache()))
	}
	if inf.Cache()[0].ID != "jira:ARO-1" {
		t.Errorf("expected cached ID jira:ARO-1, got %s", inf.Cache()[0].ID)
	}
}

func TestRelist_IncludesChildren(t *testing.T) {
	t.Parallel()
	items := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Parent", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
			Children: []*workitem.WorkItem{
				{
					TypeMeta:   workitem.TypeMeta{Kind: workitem.KindPR},
					ObjectMeta: workitem.ObjectMeta{ID: "pr:1", Label: "PR", Status: "open"},
					Spec:       json.RawMessage(`{"repo":"test","number":1}`),
				},
			},
		},
	}
	src := &staticSource{items: items}
	inf := New(logr.Discard(), src, time.Hour, time.Hour)
	h := &recordingHandler{}
	inf.RegisterHandler(h)

	inf.poll(context.Background())
	h.resetEvents()
	inf.relist()

	events := h.getEvents()
	updatedCount := 0
	for _, e := range events {
		if e.Type == EventUpdated {
			updatedCount++
		}
	}
	if updatedCount != 2 {
		t.Errorf("expected 2 Updated events from relist (parent+child), got %d", updatedCount)
	}
}
