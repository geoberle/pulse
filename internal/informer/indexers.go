package informer

import (
	"fmt"

	"github.com/geoberle/pulse/internal/workitem"
)

const (
	ByParent = "byParent"
	ByKind   = "byKind"
)

// Root items (no OwnerReferences) are indexed under the empty string key.
func parentIndexFunc(obj interface{}) ([]string, error) {
	item, ok := obj.(*workitem.WorkItem)
	if !ok {
		return nil, fmt.Errorf("expected *WorkItem, got %T", obj)
	}
	return []string{item.ParentName()}, nil
}

func kindIndexFunc(obj interface{}) ([]string, error) {
	item, ok := obj.(*workitem.WorkItem)
	if !ok {
		return nil, fmt.Errorf("expected *WorkItem, got %T", obj)
	}
	return []string{item.Kind}, nil
}
