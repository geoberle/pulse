package controller

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
)

func testLog() logr.Logger {
	return funcr.New(func(_, _ string) {}, funcr.Options{})
}

func awaitEvent[T any](t *testing.T, ch <-chan T, desc string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", desc)
	}
}

func TestController_EnqueueAndSync(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	synced := make(map[string]int)
	syncCh := make(chan string, 10)

	c := New("test", testLog(), func(_ context.Context, key string) error {
		mu.Lock()
		synced[key]++
		mu.Unlock()
		syncCh <- key
		return nil
	})

	go c.Run(ctx, 1)

	c.Enqueue("key-1")
	c.Enqueue("key-2")

	awaitEvent(t, syncCh, "key-1 sync")
	awaitEvent(t, syncCh, "key-2 sync")

	mu.Lock()
	if synced["key-1"] != 1 {
		t.Errorf("expected key-1 synced 1 time, got %d", synced["key-1"])
	}
	if synced["key-2"] != 1 {
		t.Errorf("expected key-2 synced 1 time, got %d", synced["key-2"])
	}
	mu.Unlock()
}

func TestController_ErrorRequeues(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	successCh := make(chan struct{}, 1)

	var attempts atomic.Int32

	c := New("test-retry", testLog(), func(_ context.Context, _ string) error {
		n := attempts.Add(1)
		if n <= 2 {
			return fmt.Errorf("transient error")
		}
		successCh <- struct{}{}
		return nil
	})

	go c.Run(ctx, 1)

	c.Enqueue("retry-key")

	awaitEvent(t, successCh, "successful retry")

	got := attempts.Load()
	if got < 3 {
		t.Errorf("expected at least 3 attempts (2 failures + 1 success), got %d", got)
	}
}

func TestController_Shutdown(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	c := New("test-shutdown", testLog(), func(_ context.Context, _ string) error {
		return nil
	})

	done := make(chan struct{})
	go func() {
		c.Run(ctx, 1)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("controller did not shut down within 2s")
	}
}
