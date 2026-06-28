package workitem

import "fmt"

// MakeTestItem constructs a WorkItem with a default spec for the given kind.
// Panics on invalid kind — intended for tests only.
func MakeTestItem(kind Kind, id, label string) *WorkItem {
	var spec Spec
	switch kind {
	case KindPR:
		spec = &PRSpec{Repo: "test/repo", Number: 1, Branch: "main"}
	case KindCheck:
		spec = &CheckSpec{Name: label}
	case KindReview:
		spec = &ReviewSpec{File: "test.go"}
	case KindJira:
		spec = &JiraSpec{Key: "ARO-1"}
	case KindLocal:
		spec = &LocalSpec{WorktreeID: "wt-1", Branch: "main"}
	default:
		panic(fmt.Sprintf("MakeTestItem: unknown kind %q", kind))
	}
	item, err := NewWorkItem(kind, id, label, "open", spec)
	if err != nil {
		panic(fmt.Sprintf("MakeTestItem(%q, %q): %v", kind, id, err))
	}
	return item
}
