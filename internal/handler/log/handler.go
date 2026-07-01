package log

import (
	"fmt"

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
			if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = d.Obj
			}
			logItem(log, "Deleted", obj)
		},
	}
}

func logItem(log logr.Logger, action string, obj interface{}) {
	item, ok := obj.(*workitem.WorkItem)
	if !ok {
		log.Error(nil, "unexpected type in event", "type", fmt.Sprintf("%T", obj))
		return
	}
	log.Info(action,
		"kind", item.Kind,
		"name", item.Name,
		"display", item.DisplayName(),
		"phase", item.Status.Phase,
	)
}
