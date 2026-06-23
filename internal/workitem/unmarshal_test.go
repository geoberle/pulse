package workitem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUnmarshalSpecRecursive_JiraWorkItem(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "jira_workitem.json"))
	if err != nil {
		t.Fatal(err)
	}

	var item WorkItem
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	if err := item.UnmarshalSpecRecursive(); err != nil {
		t.Fatal(err)
	}

	jiraSpec, ok := item.ParsedSpec.(*JiraSpec)
	if !ok {
		t.Fatalf("expected *JiraSpec, got %T", item.ParsedSpec)
	}
	if jiraSpec.Key != "ARO-12345" {
		t.Errorf("expected key ARO-12345, got %s", jiraSpec.Key)
	}
	if jiraSpec.Staleness != StalenessStale {
		t.Errorf("expected staleness %q, got %q", StalenessStale, jiraSpec.Staleness)
	}

	if len(item.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(item.Children))
	}
	pr := item.Children[0]
	prSpec, ok := pr.ParsedSpec.(*PRSpec)
	if !ok {
		t.Fatalf("expected *PRSpec, got %T", pr.ParsedSpec)
	}
	if prSpec.Number != 891 {
		t.Errorf("expected PR number 891, got %d", prSpec.Number)
	}
	if prSpec.Repo != "Azure/ARO-HCP" {
		t.Errorf("expected repo Azure/ARO-HCP, got %s", prSpec.Repo)
	}

	if len(pr.Children) != 2 {
		t.Fatalf("expected 2 PR children, got %d", len(pr.Children))
	}

	check := pr.Children[0]
	checkSpec, ok := check.ParsedSpec.(*CheckSpec)
	if !ok {
		t.Fatalf("expected *CheckSpec, got %T", check.ParsedSpec)
	}
	if checkSpec.Name != "e2e-test-suite" {
		t.Errorf("expected check name e2e-test-suite, got %s", checkSpec.Name)
	}

	review := pr.Children[1]
	reviewSpec, ok := review.ParsedSpec.(*ReviewSpec)
	if !ok {
		t.Fatalf("expected *ReviewSpec, got %T", review.ParsedSpec)
	}
	if reviewSpec.File != "constants.go" {
		t.Errorf("expected file constants.go, got %s", reviewSpec.File)
	}
}

func TestUnmarshalSpec_OrphanPR(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "orphan_pr.json"))
	if err != nil {
		t.Fatal(err)
	}

	var item WorkItem
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	if err := item.UnmarshalSpec(); err != nil {
		t.Fatal(err)
	}

	prSpec, ok := item.ParsedSpec.(*PRSpec)
	if !ok {
		t.Fatalf("expected *PRSpec, got %T", item.ParsedSpec)
	}
	if prSpec.Number != 910 {
		t.Errorf("expected PR number 910, got %d", prSpec.Number)
	}
	if prSpec.BranchState != BranchStateNeedsRebase {
		t.Errorf("expected branch_state %q, got %q", BranchStateNeedsRebase, prSpec.BranchState)
	}
	if len(item.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(item.Children))
	}
}

func TestUnmarshalSpec_LocalWork(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "local_work.json"))
	if err != nil {
		t.Fatal(err)
	}

	var item WorkItem
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	if err := item.UnmarshalSpec(); err != nil {
		t.Fatal(err)
	}

	localSpec, ok := item.ParsedSpec.(*LocalSpec)
	if !ok {
		t.Fatalf("expected *LocalSpec, got %T", item.ParsedSpec)
	}
	if localSpec.Branch != "experiment-branch" {
		t.Errorf("expected branch experiment-branch, got %s", localSpec.Branch)
	}
}

func TestUnmarshalSpec_UnknownKind(t *testing.T) {
	item := &WorkItem{
		TypeMeta: TypeMeta{Kind: "unknown"},
		Spec:     json.RawMessage(`{}`),
	}
	if err := item.UnmarshalSpec(); err != nil {
		t.Errorf("expected no error for unknown kind (forward-compatible), got %v", err)
	}
	if item.ParsedSpec != nil {
		t.Errorf("expected nil ParsedSpec for unknown kind, got %T", item.ParsedSpec)
	}
}

func TestUnmarshalSpec_EmptySpec(t *testing.T) {
	item := &WorkItem{
		TypeMeta: TypeMeta{Kind: KindJira},
		Spec:     json.RawMessage(`{}`),
	}
	if err := item.UnmarshalSpec(); err != nil {
		t.Fatalf("expected success for empty spec, got %v", err)
	}
	spec, ok := item.ParsedSpec.(*JiraSpec)
	if !ok {
		t.Fatalf("expected *JiraSpec, got %T", item.ParsedSpec)
	}
	if len(spec.Key) != 0 {
		t.Errorf("expected zero-valued key, got %s", spec.Key)
	}
	if spec.Staleness != StalenessUnknown {
		t.Errorf("expected zero-valued staleness, got %q", spec.Staleness)
	}
}

func TestUnmarshalSpec_ClearsStaleSpec(t *testing.T) {
	item := &WorkItem{
		TypeMeta:   TypeMeta{Kind: KindJira},
		Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		ParsedSpec: &PRSpec{Number: 999},
	}
	if err := item.UnmarshalSpec(); err != nil {
		t.Fatal(err)
	}
	if _, ok := item.ParsedSpec.(*JiraSpec); !ok {
		t.Errorf("expected *JiraSpec after re-unmarshal, got %T", item.ParsedSpec)
	}
}

func TestUnmarshalSpecRecursive_NilChild(t *testing.T) {
	item := &WorkItem{
		TypeMeta: TypeMeta{Kind: KindJira},
		Spec:     json.RawMessage(`{"key":"ARO-1"}`),
		Children: []*WorkItem{nil},
	}
	if err := item.UnmarshalSpecRecursive(); err == nil {
		t.Error("expected error for nil child")
	}
}

func TestUnmarshalSpec_MalformedJSON(t *testing.T) {
	item := &WorkItem{
		TypeMeta: TypeMeta{Kind: KindJira},
		Spec:     json.RawMessage(`{broken`),
	}
	if err := item.UnmarshalSpec(); err == nil {
		t.Error("expected error for malformed JSON spec")
	}
	if item.ParsedSpec != nil {
		t.Errorf("expected nil ParsedSpec after failed unmarshal, got %T", item.ParsedSpec)
	}
}

func TestMarshalSpec_Error(t *testing.T) {
	_, err := MarshalSpec(make(chan int))
	if err == nil {
		t.Error("expected error for unmarshalable spec")
	}
}

func TestRoundTrip_ZeroValueEnums(t *testing.T) {
	tests := []struct {
		name string
		item *WorkItem
	}{
		{
			name: "jira with zero staleness",
			item: func() *WorkItem {
				w, _ := NewWorkItem(KindJira, "jira:ARO-1", "Test", "New", &JiraSpec{
					Key:       "ARO-1",
					Staleness: StalenessUnknown,
				})
				return w
			}(),
		},
		{
			name: "pr with zero branch state",
			item: func() *WorkItem {
				w, _ := NewWorkItem(KindPR, "pr:org/repo:1", "Test PR", "open", &PRSpec{
					Repo:        "org/repo",
					Number:      1,
					Branch:      "main",
					BranchState: BranchStateUnknown,
				})
				return w
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.item)
			if err != nil {
				t.Fatal(err)
			}
			var decoded WorkItem
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if err := decoded.UnmarshalSpec(); err != nil {
				t.Fatal(err)
			}
			switch tt.item.Kind {
			case KindJira:
				spec := decoded.ParsedSpec.(*JiraSpec)
				if spec.Staleness != StalenessUnknown {
					t.Errorf("expected StalenessUnknown, got %q", spec.Staleness)
				}
			case KindPR:
				spec := decoded.ParsedSpec.(*PRSpec)
				if spec.BranchState != BranchStateUnknown {
					t.Errorf("expected BranchStateUnknown, got %q", spec.BranchState)
				}
			}
		})
	}
}

func TestUnmarshalSpec_InvalidEnum(t *testing.T) {
	item := &WorkItem{
		TypeMeta: TypeMeta{Kind: KindJira},
		Spec:     json.RawMessage(`{"key":"ARO-1","staleness":"garbage"}`),
	}
	if err := item.UnmarshalSpec(); err == nil {
		t.Error("expected error for invalid staleness enum")
	}

	item = &WorkItem{
		TypeMeta: TypeMeta{Kind: KindPR},
		Spec:     json.RawMessage(`{"repo":"org/repo","number":1,"branch":"main","branch_state":"garbage"}`),
	}
	if err := item.UnmarshalSpec(); err == nil {
		t.Error("expected error for invalid branch_state enum")
	}
}

func TestRoundTrip(t *testing.T) {
	original, err := NewWorkItem(KindJira, "jira:ARO-99999", "Test issue", "New", &JiraSpec{
		Key:       "ARO-99999",
		Staleness: StalenessActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded WorkItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.UnmarshalSpec(); err != nil {
		t.Fatal(err)
	}

	spec, ok := decoded.ParsedSpec.(*JiraSpec)
	if !ok {
		t.Fatalf("expected *JiraSpec, got %T", decoded.ParsedSpec)
	}
	if spec.Key != "ARO-99999" {
		t.Errorf("expected key ARO-99999, got %s", spec.Key)
	}
	if spec.Staleness != StalenessActive {
		t.Errorf("expected staleness %q, got %q", StalenessActive, spec.Staleness)
	}
	if decoded.ID != "jira:ARO-99999" {
		t.Errorf("expected id jira:ARO-99999, got %s", decoded.ID)
	}
}
