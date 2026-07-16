package informer

import (
	"sync"

	"github.com/geoberle/pulse/internal/api"
)

type IndexFunc[T api.Object] func(obj T) []string

type Lister[T api.Object] struct {
	mu       sync.RWMutex
	items    map[string]T
	indexers map[string]IndexFunc[T]
	indices  map[string]map[string][]string // indexName -> indexValue -> []key
}

func newLister[T api.Object]() *Lister[T] {
	return &Lister[T]{
		items:    make(map[string]T),
		indexers: make(map[string]IndexFunc[T]),
		indices:  make(map[string]map[string][]string),
	}
}

func (l *Lister[T]) List() []T {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]T, 0, len(l.items))
	for _, item := range l.items {
		if copied, ok := item.DeepCopyObject().(T); ok {
			out = append(out, copied)
		}
	}
	return out
}

func (l *Lister[T]) Get(key string) (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	item, ok := l.items[key]
	if !ok {
		var zero T
		return zero, false
	}
	if copied, ok := item.DeepCopyObject().(T); ok {
		return copied, true
	}
	var zero T
	return zero, false
}

func (l *Lister[T]) ByIndex(indexName, value string) []T {
	l.mu.RLock()
	defer l.mu.RUnlock()
	idx, ok := l.indices[indexName]
	if !ok {
		return nil
	}
	keys, ok := idx[value]
	if !ok {
		return nil
	}
	out := make([]T, 0, len(keys))
	for _, key := range keys {
		if item, exists := l.items[key]; exists {
			if copied, ok := item.DeepCopyObject().(T); ok {
				out = append(out, copied)
			}
		}
	}
	return out
}

func (l *Lister[T]) addIndexer(name string, fn IndexFunc[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indexers[name] = fn
	l.indices[name] = make(map[string][]string)
}

func (l *Lister[T]) replace(items map[string]T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = items
	l.rebuildIndicesLocked()
}

func (l *Lister[T]) rebuildIndicesLocked() {
	for name, fn := range l.indexers {
		idx := make(map[string][]string)
		for key, item := range l.items {
			for _, val := range fn(item) {
				idx[val] = append(idx[val], key)
			}
		}
		l.indices[name] = idx
	}
}
