package informer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/api"
)

type ListFunc[T api.Object] func(ctx context.Context) ([]T, error)

type Informer[T api.Object] struct {
	listFn       ListFunc[T]
	pollInterval time.Duration
	lister       *Lister[T]
	handlers     []ResourceEventHandler[T]
	synced       atomic.Bool
	log          logr.Logger
}

func New[T api.Object](log logr.Logger, listFn ListFunc[T], pollInterval time.Duration) *Informer[T] {
	return &Informer[T]{
		log:          log,
		listFn:       listFn,
		pollInterval: pollInterval,
		lister:       newLister[T](),
	}
}

func (i *Informer[T]) AddEventHandler(h ResourceEventHandler[T]) {
	i.handlers = append(i.handlers, h)
}

func (i *Informer[T]) AddIndexer(name string, fn IndexFunc[T]) {
	i.lister.addIndexer(name, fn)
}

func (i *Informer[T]) Lister() *Lister[T] {
	return i.lister
}

func (i *Informer[T]) HasSynced() bool {
	return i.synced.Load()
}

func (i *Informer[T]) Run(ctx context.Context) {
	i.poll(ctx)

	ticker := time.NewTicker(i.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.poll(ctx)
		}
	}
}

func (i *Informer[T]) poll(ctx context.Context) {
	items, err := i.listFn(ctx)
	if err != nil {
		i.log.Error(err, "failed to list")
		return
	}

	newByKey := make(map[string]T, len(items))
	for _, item := range items {
		newByKey[item.Key()] = item
	}

	oldItems := i.lister.items

	for key, newItem := range newByKey {
		oldItem, exists := oldItems[key]
		if !exists {
			copied, ok := newItem.DeepCopyObject().(T)
			if !ok {
				i.log.Error(nil, "type assertion failed on DeepCopyObject", "key", key)
				continue
			}
			i.dispatch(func(h ResourceEventHandler[T]) {
				h.OnAdd(copied)
			})
			continue
		}
		if oldItem.GetObjectMeta().ResourceVersion != newItem.GetObjectMeta().ResourceVersion {
			oldCopy, ok := oldItem.DeepCopyObject().(T)
			if !ok {
				i.log.Error(nil, "type assertion failed on DeepCopyObject", "key", key)
				continue
			}
			newCopy, ok := newItem.DeepCopyObject().(T)
			if !ok {
				i.log.Error(nil, "type assertion failed on DeepCopyObject", "key", key)
				continue
			}
			i.dispatch(func(h ResourceEventHandler[T]) {
				h.OnUpdate(oldCopy, newCopy)
			})
		}
	}

	for key, oldItem := range oldItems {
		if _, exists := newByKey[key]; !exists {
			copied, ok := oldItem.DeepCopyObject().(T)
			if !ok {
				i.log.Error(nil, "type assertion failed on DeepCopyObject", "key", key)
				continue
			}
			i.dispatch(func(h ResourceEventHandler[T]) {
				h.OnDelete(copied)
			})
		}
	}

	i.lister.replace(newByKey)
	i.synced.Store(true)
}

func (i *Informer[T]) dispatch(fn func(ResourceEventHandler[T])) {
	for _, h := range i.handlers {
		fn(h)
	}
}
