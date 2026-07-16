package controller

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller interface {
	QueueForInformers(resyncDuration time.Duration, notifiers ...cache.SharedIndexInformer) error
	SyncOnce(ctx context.Context, key string) error
	Run(ctx context.Context, workers int)
}

type SyncFunc func(ctx context.Context, key string) error

type BaseController struct {
	name   string
	queue  workqueue.TypedRateLimitingInterface[string]
	syncFn SyncFunc
	log    logr.Logger
}

func New(name string, log logr.Logger, syncFn SyncFunc) *BaseController {
	return &BaseController{
		name: name,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: name},
		),
		syncFn: syncFn,
		log:    log.WithName(name),
	}
}

func (c *BaseController) QueueForInformers(resyncDuration time.Duration, notifiers ...cache.SharedIndexInformer) error {
	for _, notifier := range notifiers {
		_, err := notifier.AddEventHandlerWithOptions(
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
						c.queue.Add(key)
					}
				},
				UpdateFunc: func(_, obj interface{}) {
					if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
						c.queue.Add(key)
					}
				},
				DeleteFunc: func(obj interface{}) {
					if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
						c.queue.Add(key)
					}
				},
			},
			cache.HandlerOptions{ResyncPeriod: &resyncDuration},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *BaseController) SyncOnce(ctx context.Context, key string) error {
	return c.syncFn(ctx, key)
}

func (c *BaseController) Enqueue(key string) {
	c.queue.Add(key)
}

func (c *BaseController) EnqueueAfter(key string, delay time.Duration) {
	c.queue.AddAfter(key, delay)
}

func (c *BaseController) Run(ctx context.Context, workers int) {
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

func (c *BaseController) processNext(ctx context.Context) bool {
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
