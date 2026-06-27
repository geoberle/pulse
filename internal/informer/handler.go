package informer

import "github.com/geoberle/pulse/internal/workitem"

type EventType string

const (
	EventAdded   EventType = "Added"
	EventUpdated EventType = "Updated"
	EventDeleted EventType = "Deleted"
)

type Event struct {
	Type   EventType
	Old    *workitem.WorkItem
	New    *workitem.WorkItem
	Parent *workitem.WorkItem
}

type Handler interface {
	OnEvent(Event)
}
