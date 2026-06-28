package informer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/workitem"
)

// Source provides the current set of work items. Implementations must not
// return nil elements in the slice or in Children.
type Source interface {
	List(ctx context.Context) ([]*workitem.WorkItem, error)
}

var _ Lister = (*Informer)(nil)

type Informer struct {
	source         Source
	store          Store
	pollInterval   time.Duration
	relistInterval time.Duration
	cache          []*workitem.WorkItem
	handlers       []Handler
	synced         atomic.Bool
	log            logr.Logger
}

type Options struct {
	Store          Store
	Initial        []*workitem.WorkItem
	PollInterval   time.Duration
	RelistInterval time.Duration
}

func New(log logr.Logger, source Source, opts Options) *Informer {
	inf := &Informer{
		log:            log,
		source:         source,
		store:          opts.Store,
		pollInterval:   opts.PollInterval,
		relistInterval: opts.RelistInterval,
		cache:          opts.Initial,
	}
	if len(opts.Initial) > 0 {
		inf.synced.Store(true)
	}
	return inf
}

// RegisterHandler adds a handler. Must be called before Run.
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
		inf.log.Error(err, "failed to list work items")
		return
	}
	events := diffTrees(inf.cache, items, nil)
	inf.cache = items
	inf.synced.Store(true)
	if inf.store != nil {
		if err := inf.store.Save(items); err != nil {
			inf.log.Error(err, "failed to persist state")
		}
	}
	inf.dispatch(events)
}

func (inf *Informer) relist() {
	var events []Event
	walkTree(inf.cache, nil, func(item, parent *workitem.WorkItem) {
		events = append(events, Event{Type: EventUpdated, New: item, Parent: parent})
	})
	inf.dispatch(events)
}

func walkTree(items []*workitem.WorkItem, parent *workitem.WorkItem, fn func(*workitem.WorkItem, *workitem.WorkItem)) {
	for _, item := range items {
		fn(item, parent)
		walkTree(item.Children, item, fn)
	}
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

// List returns the current cached items, satisfying the Lister interface.
// The returned slice and its elements must not be mutated. Must not be
// called concurrently with Run.
func (inf *Informer) List() []*workitem.WorkItem {
	return inf.cache
}
