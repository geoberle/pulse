package workitem

import "encoding/json"

const (
	KindJira   = "jira"
	KindPR     = "pr"
	KindCheck  = "check"
	KindReview = "review"
	KindLocal  = "local"
)

type TypeMeta struct {
	Kind string `json:"kind"`
}

type ObjectMeta struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
}

type WorkItem struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:",inline"`
	Spec       json.RawMessage `json:"spec,omitempty"`
	Children   []*WorkItem     `json:"children,omitempty"`
	ParsedSpec any             `json:"-"`
}
