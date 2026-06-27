package informer

import (
	"bytes"
	"encoding/json"

	"github.com/geoberle/pulse/internal/workitem"
)

func canonicalizeJSON(raw json.RawMessage) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return canonical
}

func itemChanged(a, b *workitem.WorkItem) bool {
	return a.Kind != b.Kind || a.ID != b.ID || a.Label != b.Label ||
		a.Status != b.Status ||
		!bytes.Equal(canonicalizeJSON(a.Spec), canonicalizeJSON(b.Spec))
}

func diffTrees(oldItems, newItems []*workitem.WorkItem, parent *workitem.WorkItem) []Event {
	oldByID := make(map[string]*workitem.WorkItem, len(oldItems))
	for _, item := range oldItems {
		if item == nil {
			continue
		}
		oldByID[item.ID] = item
	}

	newByID := make(map[string]*workitem.WorkItem, len(newItems))
	for _, item := range newItems {
		if item == nil {
			continue
		}
		newByID[item.ID] = item
	}

	var events []Event

	for _, newItem := range newItems {
		if newItem == nil {
			continue
		}
		oldItem, exists := oldByID[newItem.ID]
		if !exists {
			events = append(events, Event{Type: EventAdded, New: newItem, Parent: parent})
			events = append(events, diffTrees(nil, newItem.Children, newItem)...)
			continue
		}
		if itemChanged(oldItem, newItem) {
			events = append(events, Event{Type: EventUpdated, Old: oldItem, New: newItem, Parent: parent})
		}
		events = append(events, diffTrees(oldItem.Children, newItem.Children, newItem)...)
	}

	for _, oldItem := range oldItems {
		if oldItem == nil {
			continue
		}
		if _, exists := newByID[oldItem.ID]; !exists {
			events = append(events, diffTrees(oldItem.Children, nil, oldItem)...)
			events = append(events, Event{Type: EventDeleted, Old: oldItem, Parent: parent})
		}
	}

	return events
}
