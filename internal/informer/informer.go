package informer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/geoberle/pulse/internal/workitem"
)

// Source provides the current set of work items. Implementations fetch from
// external systems (Jira, GitHub, etc.).
type Source interface {
	List(ctx context.Context) ([]*workitem.WorkItem, error)
}

type Informer struct {
	source         Source
	pollInterval   time.Duration
	relistInterval time.Duration
	cache          []*workitem.WorkItem
	handlers       []Handler
	synced         atomic.Bool
}

func New(source Source, pollInterval, relistInterval time.Duration) *Informer {
	return &Informer{
		source:         source,
		pollInterval:   pollInterval,
		relistInterval: relistInterval,
	}
}

func (inf *Informer) RegisterHandler(h Handler) {
	inf.handlers = append(inf.handlers, h)
}

// Run polls the source at pollInterval, diffs against the cache, and dispatches
// events for changes. At relistInterval, all cached items are re-delivered as
// EventUpdated regardless of changes. Blocks until ctx is cancelled.
func (inf *Informer) Run(ctx context.Context) {
	pollTicker := time.NewTicker(inf.pollInterval)
	defer pollTicker.Stop()
	relistTicker := time.NewTicker(inf.relistInterval)
	defer relistTicker.Stop()

	inf.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-relistTicker.C:
			inf.relist()
		case <-pollTicker.C:
			inf.poll(ctx)
		}
	}
}

func (inf *Informer) poll(ctx context.Context) {
	items, err := inf.source.List(ctx)
	if err != nil {
		return
	}
	events := diffTrees(inf.cache, items, nil)
	inf.cache = items
	inf.synced.Store(true)
	inf.dispatch(events)
}

func (inf *Informer) relist() {
	var events []Event
	for _, item := range inf.cache {
		if item == nil {
			continue
		}
		events = append(events, Event{Type: EventUpdated, New: item})
		events = append(events, relistChildren(item.Children, item)...)
	}
	inf.dispatch(events)
}

func relistChildren(items []*workitem.WorkItem, parent *workitem.WorkItem) []Event {
	var events []Event
	for _, item := range items {
		if item == nil {
			continue
		}
		events = append(events, Event{Type: EventUpdated, New: item, Parent: parent})
		events = append(events, relistChildren(item.Children, item)...)
	}
	return events
}

func (inf *Informer) dispatch(events []Event) {
	for _, e := range events {
		for _, h := range inf.handlers {
			h.OnEvent(e)
		}
	}
}

func (inf *Informer) HasSynced() bool {
	return inf.synced.Load()
}

// Cache returns the current cached items. The returned slice and its elements
// must not be mutated.
func (inf *Informer) Cache() []*workitem.WorkItem {
	return inf.cache
}
