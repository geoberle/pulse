package workitem

import (
	"encoding/json"
	"fmt"
)

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
	ID string `json:"id"`

	// Label is a short human-readable display string for TUI rendering,
	// e.g. the Jira summary or PR title.
	Label string `json:"label"`

	// Status is the upstream state of the item (e.g. "In Progress", "open",
	// "failed", "pending"). Semantics are kind-specific.
	Status string `json:"status"`
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
	ParsedSpec any `json:"-"`
}
