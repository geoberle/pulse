package informer

import (
	"fmt"

	"github.com/geoberle/pulse/internal/workitem"
)

const (
	ByParent = "byParent"
	ByKind   = "byKind"
)

func ParentIndexFunc(obj interface{}) ([]string, error) {
	item, ok := obj.(*workitem.WorkItem)
	if !ok {
		return nil, fmt.Errorf("expected *WorkItem, got %T", obj)
	}
	return []string{item.ParentID}, nil
}

func KindIndexFunc(obj interface{}) ([]string, error) {
	item, ok := obj.(*workitem.WorkItem)
	if !ok {
		return nil, fmt.Errorf("expected *WorkItem, got %T", obj)
	}
	return []string{string(item.Kind)}, nil
}
