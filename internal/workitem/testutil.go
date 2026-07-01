package workitem

import "fmt"

// MakeTestItem constructs a WorkItem with a default spec for the given kind.
// Panics on invalid kind — intended for tests only.
func MakeTestItem(kind Kind, name string) *WorkItem {
	var spec Spec
	switch kind {
	case KindPR:
		spec = &PRSpec{Repo: "test/repo", Number: 1, Branch: "main", Title: "Test PR"}
	case KindCheck:
		spec = &CheckSpec{Name: "ci"}
	case KindReview:
		spec = &ReviewSpec{File: "test.go"}
	case KindJira:
		spec = &JiraSpec{Key: "ARO-1", Summary: "Test issue"}
	case KindLocal:
		spec = &LocalSpec{WorktreeID: "wt-1", Branch: "main"}
	default:
		panic(fmt.Sprintf("MakeTestItem: unknown kind %q", kind))
	}
	item, err := NewWorkItem(kind, name, "open", spec)
	if err != nil {
		panic(fmt.Sprintf("MakeTestItem(%q, %q): %v", kind, name, err))
	}
	return item
}
