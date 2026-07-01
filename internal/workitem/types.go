package workitem

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Spec is implemented by all kind-specific spec structs.
// DeepCopySpec is required by deepcopy-gen for interface-typed fields.
type Spec interface {
	Validate() error
	DeepCopySpec() Spec
}

const APIVersion = "pulse.dev/v1"

// Kind identifies the type of a WorkItem. Each kind maps to a
// corresponding Spec struct (e.g. KindJira → JiraSpec).
type Kind string

const (
	KindJira   Kind = "jira"
	KindPR     Kind = "pr"
	KindCheck  Kind = "check"
	KindReview Kind = "review"
	KindLocal  Kind = "local"
)

// Validate checks that the Kind is registered in specFactories.
func (k Kind) Validate() error {
	if _, ok := specFactories[k]; !ok {
		return fmt.Errorf("unknown kind %q", k)
	}
	return nil
}

// WorkItemPhase is the primary display status of a WorkItem. Values are
// kind-specific and come from upstream APIs (e.g. GitHub PR state, Jira
// status name, check conclusion).
type WorkItemPhase string

// WorkItemStatus holds lifecycle state and orthogonal condition signals.
type WorkItemStatus struct {
	Phase      WorkItemPhase      `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// WorkItem is the unified resource for all dashboard entities. It follows
// the Kubernetes resource shape: TypeMeta + ObjectMeta + Spec + Status.
// Children form a recursive tree: Jira → PRs → Checks/Reviews.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the kind-specific payload as raw JSON. Deserialized into
	// a typed struct (JiraSpec, PRSpec, etc.) via UnmarshalSpec and stored
	// in ParsedSpec.
	Spec json.RawMessage `json:"spec,omitempty"`

	// Status holds lifecycle phase and orthogonal condition signals.
	Status WorkItemStatus `json:"status,omitempty"`

	// Children are nested WorkItems forming the display tree. A Jira item
	// has PR children, a PR has Check and Review children.
	Children []*WorkItem `json:"children,omitempty"`

	// ParsedSpec holds the deserialized Spec after UnmarshalSpec is called.
	// Not serialized — populated at runtime only. Exported for deepcopy-gen
	// and read access by consumers (e.g. DisplayName type switch). Do not
	// set directly outside the workitem package; use UnmarshalSpec or NewWorkItem.
	ParsedSpec Spec `json:"-"`
}

// DisplayName returns the human-readable display string for this item,
// extracted from the kind-specific Spec fields. Falls back to Name
// (the cache key) when ParsedSpec is nil (unknown kinds from disk).
func (w *WorkItem) DisplayName() string {
	if w.ParsedSpec == nil {
		return w.Name
	}
	switch s := w.ParsedSpec.(type) {
	case *JiraSpec:
		return s.Summary
	case *PRSpec:
		return s.Title
	case *CheckSpec:
		return s.Name
	case *ReviewSpec:
		return s.File
	case *LocalSpec:
		return s.Branch
	default:
		return w.Name
	}
}

// ParentName returns the Name of this item's parent from OwnerReferences,
// or "" if the item is a root (no parent).
func (w *WorkItem) ParentName() string {
	if len(w.OwnerReferences) == 0 {
		return ""
	}
	return w.OwnerReferences[0].Name
}

// WorkItemList is a list of WorkItems compatible with runtime.Object
// for use with Kubernetes informer machinery.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkItem `json:"items"`
}
