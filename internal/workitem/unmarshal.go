package workitem

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (w *WorkItem) UnmarshalSpec() error {
	w.ParsedSpec = nil
	if len(w.Spec) == 0 {
		return nil
	}
	factory, ok := specFactories[Kind(w.Kind)]
	if !ok {
		return nil
	}
	spec := factory()
	if err := json.Unmarshal(w.Spec, spec); err != nil {
		return fmt.Errorf("unmarshal spec for kind %s: %w", w.Kind, err)
	}
	w.ParsedSpec = spec
	return nil
}

func (w *WorkItem) UnmarshalSpecRecursive() error {
	if err := w.UnmarshalSpec(); err != nil {
		return err
	}
	for i, child := range w.Children {
		if child == nil {
			return fmt.Errorf("nil child at index %d", i)
		}
		if err := child.UnmarshalSpecRecursive(); err != nil {
			return err
		}
	}
	return nil
}

func MarshalSpec(spec Spec) (json.RawMessage, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec of type %T: %w", spec, err)
	}
	return json.RawMessage(data), nil
}

// NewWorkItem constructs a WorkItem with a validated Kind and marshaled Spec.
// Spec is marshaled to JSON immediately — mutations after this call do not
// affect the WorkItem's Spec field.
func NewWorkItem(kind Kind, name string, phase WorkItemPhase, spec Spec) (*WorkItem, error) {
	if err := kind.Validate(); err != nil {
		return nil, err
	}
	if len(name) == 0 {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > 500 {
		return nil, fmt.Errorf("name: max 500 chars, got %d", len(name))
	}
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("validate spec for kind %s: %w", kind, err)
	}
	raw, err := MarshalSpec(spec)
	if err != nil {
		return nil, err
	}
	return &WorkItem{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(kind),
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       raw,
		Status:     WorkItemStatus{Phase: phase},
		ParsedSpec: spec,
	}, nil
}
