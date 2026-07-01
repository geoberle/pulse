package informer

import (
	"fmt"

	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/workitem"
)

// Lister provides read access to the informer's indexed cache.
// Returned items are shared pointers into the cache — callers must not mutate them.
type Lister interface {
	List() ([]*workitem.WorkItem, error)
	Get(id string) (*workitem.WorkItem, bool, error)
	ByIndex(indexName, key string) ([]*workitem.WorkItem, error)
}

type lister struct {
	indexer cache.Indexer
}

// NewLister creates a Lister backed by the given cache.Indexer.
func NewLister(indexer cache.Indexer) Lister {
	return &lister{indexer: indexer}
}

func (l *lister) List() ([]*workitem.WorkItem, error) {
	objs := l.indexer.List()
	items := make([]*workitem.WorkItem, 0, len(objs))
	for _, obj := range objs {
		item, ok := obj.(*workitem.WorkItem)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T in cache", obj)
		}
		items = append(items, item)
	}
	return items, nil
}

func (l *lister) Get(id string) (*workitem.WorkItem, bool, error) {
	obj, exists, err := l.indexer.GetByKey(id)
	if err != nil || !exists {
		return nil, exists, err
	}
	item, ok := obj.(*workitem.WorkItem)
	if !ok {
		return nil, false, fmt.Errorf("unexpected type %T in cache", obj)
	}
	return item, true, nil
}

func (l *lister) ByIndex(indexName, key string) ([]*workitem.WorkItem, error) {
	objs, err := l.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	items := make([]*workitem.WorkItem, 0, len(objs))
	for _, obj := range objs {
		item, ok := obj.(*workitem.WorkItem)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T in cache", obj)
		}
		items = append(items, item)
	}
	return items, nil
}
