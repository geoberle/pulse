package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/informer"
	"github.com/geoberle/pulse/internal/poller"
	"github.com/geoberle/pulse/internal/workitem"
)

var _ informer.Source = (*Engine)(nil)

type Engine struct {
	pollers    []poller.Poller
	log        logr.Logger
	mu         sync.Mutex
	lastErrors []error
}

func New(log logr.Logger, pollers []poller.Poller) *Engine {
	return &Engine{
		log:     log,
		pollers: pollers,
	}
}

func (e *Engine) List(ctx context.Context) ([]*workitem.WorkItem, error) {
	var allItems []*workitem.WorkItem
	var errs []error

	for i, p := range e.pollers {
		items, err := p.Poll(ctx)
		if err != nil {
			e.log.Error(err, "poller failed", "index", i)
			errs = append(errs, fmt.Errorf("poller %d: %w", i, err))
			continue
		}
		allItems = append(allItems, items...)
	}

	e.mu.Lock()
	e.lastErrors = errs
	e.mu.Unlock()

	if len(allItems) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all pollers failed: %d errors", len(errs))
	}

	return merge(allItems), nil
}

// Errors returns errors from the last List call. Safe for concurrent use.
func (e *Engine) Errors() []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastErrors
}
