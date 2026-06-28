package log

import (
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/workitem"
)

func NewHandler(log logr.Logger) cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logItem(log, "Added", obj)
		},
		UpdateFunc: func(_, newObj interface{}) {
			logItem(log, "Updated", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			logItem(log, "Deleted", obj)
		},
	}
}

func logItem(log logr.Logger, action string, obj interface{}) {
	item, ok := obj.(*workitem.WorkItem)
	if !ok {
		return
	}
	log.Info(action,
		"kind", string(item.Kind),
		"id", item.ID,
		"label", item.Label,
		"status", item.Status,
	)
}
