package workitem

import (
	"encoding/json"
	"fmt"
)

func (w *WorkItem) UnmarshalSpec() error {
	w.ParsedSpec = nil
	if len(w.Spec) == 0 {
		return nil
	}
	var spec any
	switch w.Kind {
	case KindJira:
		spec = &JiraSpec{}
	case KindPR:
		spec = &PRSpec{}
	case KindCheck:
		spec = &CheckSpec{}
	case KindReview:
		spec = &ReviewSpec{}
	case KindLocal:
		spec = &LocalSpec{}
	default:
		return nil
	}
	if err := json.Unmarshal(w.Spec, spec); err != nil {
		return fmt.Errorf("unmarshal spec for kind %s: %w", w.Kind, err)
	}
	if v, ok := spec.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("validate spec for kind %s: %w", w.Kind, err)
		}
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

func MarshalSpec(spec any) (json.RawMessage, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec of type %T: %w", spec, err)
	}
	return json.RawMessage(data), nil
}

func NewWorkItem(kind, id, label, status string, spec any) (*WorkItem, error) {
	raw, err := MarshalSpec(spec)
	if err != nil {
		return nil, err
	}
	return &WorkItem{
		TypeMeta:   TypeMeta{Kind: kind},
		ObjectMeta: ObjectMeta{ID: id, Label: label, Status: status},
		Spec:       raw,
		ParsedSpec: spec,
	}, nil
}
