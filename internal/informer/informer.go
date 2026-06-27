package informer

import "github.com/geoberle/pulse/internal/workitem"

type Informer struct {
	cache    []*workitem.WorkItem
	handlers []Handler
}

func New() *Informer {
	return &Informer{}
}

func (inf *Informer) RegisterHandler(h Handler) {
	inf.handlers = append(inf.handlers, h)
}

func (inf *Informer) Sync(items []*workitem.WorkItem) {
	events := diffTrees(inf.cache, items, nil)
	inf.cache = items
	for _, e := range events {
		for _, h := range inf.handlers {
			h.OnEvent(e)
		}
	}
}

func (inf *Informer) Cache() []*workitem.WorkItem {
	return inf.cache
}
