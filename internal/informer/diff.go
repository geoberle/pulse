package informer

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/geoberle/pulse/internal/workitem"
)

func hashItem(item *workitem.WorkItem) string {
	h := sha256.New()
	h.Write([]byte(item.Kind))
	h.Write([]byte{0})
	h.Write([]byte(item.ID))
	h.Write([]byte{0})
	h.Write([]byte(item.Label))
	h.Write([]byte{0})
	h.Write([]byte(item.Status))
	h.Write([]byte{0})
	h.Write(item.Spec)
	return hex.EncodeToString(h.Sum(nil))
}

func diffTrees(oldItems, newItems []*workitem.WorkItem, parent *workitem.WorkItem) []Event {
	oldByID := make(map[string]*workitem.WorkItem, len(oldItems))
	for _, item := range oldItems {
		oldByID[item.ID] = item
	}

	newByID := make(map[string]*workitem.WorkItem, len(newItems))
	for _, item := range newItems {
		newByID[item.ID] = item
	}

	var events []Event

	for _, newItem := range newItems {
		oldItem, exists := oldByID[newItem.ID]
		if !exists {
			events = append(events, Event{Type: EventAdded, New: newItem, Parent: parent})
			events = append(events, diffTrees(nil, newItem.Children, newItem)...)
			continue
		}
		if hashItem(oldItem) != hashItem(newItem) {
			events = append(events, Event{Type: EventUpdated, Old: oldItem, New: newItem, Parent: parent})
		}
		events = append(events, diffTrees(oldItem.Children, newItem.Children, newItem)...)
	}

	for _, oldItem := range oldItems {
		if _, exists := newByID[oldItem.ID]; !exists {
			events = append(events, diffTrees(oldItem.Children, nil, oldItem)...)
			events = append(events, Event{Type: EventDeleted, Old: oldItem, Parent: parent})
		}
	}

	return events
}
