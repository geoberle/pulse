package log

import (
	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/informer"
)

var _ informer.Handler = (*Handler)(nil)

type Handler struct {
	log logr.Logger
}

func NewHandler(log logr.Logger) *Handler {
	return &Handler{log: log}
}

func (h *Handler) OnEvent(e informer.Event) {
	item := e.New
	if item == nil {
		item = e.Old
	}
	kv := []any{
		"kind", string(item.Kind),
		"id", item.ID,
		"label", item.Label,
		"status", item.Status,
	}
	if e.Parent != nil {
		kv = append(kv, "parent", e.Parent.ID)
	}
	h.log.Info(string(e.Type), kv...)
}
