package workitem

import (
	"encoding/json"
	"fmt"
)

// Spec is implemented by all kind-specific spec structs.
type Spec interface {
	Validate() error
}

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

// Validate checks that the Kind is a known value.
func (k Kind) Validate() error {
	switch k {
	case KindJira, KindPR, KindCheck, KindReview, KindLocal:
		return nil
	default:
		return fmt.Errorf("unknown kind %q", k)
	}
}

// TypeMeta identifies the kind of a WorkItem. Mirrors the Kubernetes
// TypeMeta pattern — every serialized WorkItem carries its kind so the
// correct Spec struct can be selected during unmarshal.
type TypeMeta struct {
	// Kind is one of the Kind* constants (e.g. "jira", "pr", "check").
	Kind Kind `json:"kind"`
}

// ObjectMeta holds identity and display fields common to every WorkItem,
// regardless of kind. Mirrors Kubernetes ObjectMeta at a minimal scope.
type ObjectMeta struct {
	// ID uniquely identifies the item using the pattern "{source}:{identifier}",
	// e.g. "jira:ARO-12345", "pr:Azure/ARO-HCP:891", "gh-comment:3453365398".
	// Maximum 500 characters.
	ID string `json:"id"`

	// Label is a short human-readable display string for TUI rendering,
	// e.g. the Jira summary or PR title. Maximum 1000 characters.
	Label string `json:"label"`

	// Status is the upstream state of the item (e.g. "In Progress", "open",
	// "failed", "pending"). Semantics are kind-specific. Maximum 500 characters.
	Status string `json:"status"`
}

// Validate checks length constraints on ObjectMeta fields.
func (m *ObjectMeta) Validate() error {
	if len(m.ID) > 500 {
		return fmt.Errorf("id: max 500 chars, got %d", len(m.ID))
	}
	if len(m.Label) > 1000 {
		return fmt.Errorf("label: max 1000 chars, got %d", len(m.Label))
	}
	if len(m.Status) > 500 {
		return fmt.Errorf("status: max 500 chars, got %d", len(m.Status))
	}
	return nil
}

// WorkItem is the unified tree node for all dashboard entities. It combines
// common metadata (TypeMeta + ObjectMeta) with a kind-specific Spec stored
// as json.RawMessage for lazy deserialization. Children form a recursive
// tree: Jira → PRs → Checks/Reviews.
type WorkItem struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:",inline"`

	// Spec holds the kind-specific payload as raw JSON. Deserialized into
	// a typed struct (JiraSpec, PRSpec, etc.) via UnmarshalSpec and stored
	// in ParsedSpec.
	Spec json.RawMessage `json:"spec,omitempty"`

	// Children are nested WorkItems forming the display tree. A Jira item
	// has PR children, a PR has Check and Review children.
	Children []*WorkItem `json:"children,omitempty"`

	// ParsedSpec holds the deserialized Spec after UnmarshalSpec is called.
	// Not serialized — populated at runtime only.
	ParsedSpec Spec `json:"-"`
}
