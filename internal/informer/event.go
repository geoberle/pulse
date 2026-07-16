package informer

import "github.com/geoberle/pulse/internal/api"

type ResourceEventHandler[T api.Object] interface {
	OnAdd(obj T)
	OnUpdate(oldObj, newObj T)
	OnDelete(obj T)
}

type ResourceEventHandlerFuncs[T api.Object] struct {
	AddFunc    func(obj T)
	UpdateFunc func(oldObj, newObj T)
	DeleteFunc func(obj T)
}

func (h ResourceEventHandlerFuncs[T]) OnAdd(obj T) {
	if h.AddFunc != nil {
		h.AddFunc(obj)
	}
}

func (h ResourceEventHandlerFuncs[T]) OnUpdate(oldObj, newObj T) {
	if h.UpdateFunc != nil {
		h.UpdateFunc(oldObj, newObj)
	}
}

func (h ResourceEventHandlerFuncs[T]) OnDelete(obj T) {
	if h.DeleteFunc != nil {
		h.DeleteFunc(obj)
	}
}
