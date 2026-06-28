package informer

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/workitem"
)

// Source provides the current set of work items as a tree. The informer
// flattens the tree before storing items in the cache.
type Source interface {
	List(ctx context.Context) ([]*workitem.WorkItem, error)
}

// New creates a SharedIndexInformer that polls source at pollInterval.
// The source returns a tree; the informer flattens it for storage.
// Items are indexed by ByParent and ByKind for tree reconstruction.
func New(source Source, pollInterval time.Duration) cache.SharedIndexInformer {
	lw := &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, _ metav1.ListOptions) (runtime.Object, error) {
			items, err := source.List(ctx)
			if err != nil {
				return nil, err
			}
			flat := workitem.Flatten(items)
			return &workitem.WorkItemList{Items: flat}, nil
		},
		WatchFuncWithContext: func(ctx context.Context, _ metav1.ListOptions) (watch.Interface, error) {
			return newExpiringWatcher(ctx, pollInterval), nil
		},
	}

	return cache.NewSharedIndexInformerWithOptions(
		&listWatchWithoutWatchListSemantics{lw},
		&workitem.WorkItem{},
		cache.SharedIndexInformerOptions{
			ResyncPeriod: 0,
			Indexers: cache.Indexers{
				ByParent: ParentIndexFunc,
				ByKind:   KindIndexFunc,
			},
		},
	)
}
