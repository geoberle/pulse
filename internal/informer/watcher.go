package informer

import (
	"context"
	"net/http"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// listWatchWithoutWatchListSemantics opts out of WatchListClient
// semantics. Our ExpiringWatcher does not support the bookmark
// protocol that WatchListClient requires.
type listWatchWithoutWatchListSemantics struct {
	*cache.ListWatch
}

func (listWatchWithoutWatchListSemantics) IsWatchListSemanticsUnSupported() bool { return true }

type expiringWatcher struct {
	result chan watch.Event
	done   chan struct{}
	stop   sync.Once
}

// newExpiringWatcher creates a watcher that sends an HTTP 410 Gone
// error after the given duration, causing the reflector to relist.
func newExpiringWatcher(ctx context.Context, expiry time.Duration) watch.Interface {
	w := &expiringWatcher{
		result: make(chan watch.Event),
		done:   make(chan struct{}),
	}
	go func() {
		defer utilruntime.HandleCrash()
		timer := time.NewTimer(expiry)
		defer timer.Stop()
		select {
		case <-timer.C:
			w.result <- watch.Event{
				Type: watch.Error,
				Object: &metav1.Status{
					Status:  metav1.StatusFailure,
					Code:    http.StatusGone,
					Reason:  metav1.StatusReasonExpired,
					Message: "watch expired",
				},
			}
		case <-w.done:
		case <-ctx.Done():
		}
		close(w.result)
	}()
	return w
}

func (w *expiringWatcher) Stop() {
	w.stop.Do(func() { close(w.done) })
}

func (w *expiringWatcher) ResultChan() <-chan watch.Event { return w.result }
