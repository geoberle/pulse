package informer

import "github.com/geoberle/pulse/internal/workitem"

type Lister interface {
	List() []*workitem.WorkItem
}
