package informer

import "github.com/geoberle/pulse/internal/workitem"

type Store interface {
	Save(items []*workitem.WorkItem) error
	Load() ([]*workitem.WorkItem, error)
}
