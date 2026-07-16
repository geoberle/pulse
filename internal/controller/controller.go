package controller

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
)

type SyncFunc func(ctx context.Context, key string) error

type Controller struct {
	name   string
	queue  workqueue.TypedRateLimitingInterface[string]
	syncFn SyncFunc
	log    logr.Logger
}

func New(name string, log logr.Logger, syncFn SyncFunc) *Controller {
	return &Controller{
		name: name,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: name},
		),
		syncFn: syncFn,
		log:    log.WithName(name),
	}
}

func (c *Controller) Enqueue(key string) {
	c.queue.Add(key)
}

func (c *Controller) Run(ctx context.Context, workers int) {
	c.log.Info("starting controller", "workers", workers)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c.processNext(ctx) {
			}
		}()
	}

	<-ctx.Done()
	c.log.Info("shutting down controller")
	c.queue.ShutDown()
	wg.Wait()
}

func (c *Controller) processNext(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	if err := c.syncFn(ctx, key); err != nil {
		c.log.Error(err, "sync failed, requeuing", "key", key)
		c.queue.AddRateLimited(key)
		return true
	}

	c.queue.Forget(key)
	return true
}

func (c *Controller) EnqueueAfter(key string, delay time.Duration) {
	c.queue.AddAfter(key, delay)
}
