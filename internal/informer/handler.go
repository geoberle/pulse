package informer

import (
	"fmt"

	"github.com/geoberle/pulse/internal/workitem"
)

type EventType string

const (
	EventAdded   EventType = "Added"
	EventUpdated EventType = "Updated"
	EventDeleted EventType = "Deleted"
)

func (e EventType) Validate() error {
	switch e {
	case EventAdded, EventUpdated, EventDeleted:
		return nil
	default:
		return fmt.Errorf("unknown event type %q", e)
	}
}

type Event struct {
	Type   EventType
	Old    *workitem.WorkItem
	New    *workitem.WorkItem
	Parent *workitem.WorkItem
}

type Handler interface {
	OnEvent(Event)
}
