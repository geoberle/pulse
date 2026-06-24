package workitem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUnmarshalSpecRecursive_JiraWorkItem(t *testing.T) {
	t.Parallel()
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

func TestUnmarshalSpec_GoldenFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		file    string
		checkFn func(t *testing.T, item *WorkItem)
	}{
		{
			name: "orphan PR",
			file: "orphan_pr.json",
			checkFn: func(t *testing.T, item *WorkItem) {
				spec, ok := item.ParsedSpec.(*PRSpec)
				if !ok {
					t.Fatalf("expected *PRSpec, got %T", item.ParsedSpec)
				}
				if spec.Repo != "Azure/ARO-HCP" {
					t.Errorf("expected repo Azure/ARO-HCP, got %s", spec.Repo)
				}
				if spec.Number != 910 {
					t.Errorf("expected number 910, got %d", spec.Number)
				}
				if spec.BranchState != BranchStateNeedsRebase {
					t.Errorf("expected branch_state %q, got %q", BranchStateNeedsRebase, spec.BranchState)
				}
			},
		},
		{
			name: "local work",
			file: "local_work.json",
			checkFn: func(t *testing.T, item *WorkItem) {
				spec, ok := item.ParsedSpec.(*LocalSpec)
				if !ok {
					t.Fatalf("expected *LocalSpec, got %T", item.ParsedSpec)
				}
				if spec.Branch != "experiment-branch" {
					t.Errorf("expected branch experiment-branch, got %s", spec.Branch)
				}
				if len(spec.WorktreeID) == 0 {
					t.Error("expected non-empty worktree_id")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join("testdata", tt.file))
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
			if item.ParsedSpec == nil {
				t.Fatal("expected non-nil ParsedSpec")
			}
			tt.checkFn(t, &item)
		})
	}
}

func TestUnmarshalSpec_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		item    *WorkItem
		wantErr bool
		checkFn func(t *testing.T, item *WorkItem)
	}{
		{
			name: "unknown kind (forward-compatible)",
			item: &WorkItem{
				TypeMeta: TypeMeta{Kind: "unknown"},
				Spec:     json.RawMessage(`{}`),
			},
			checkFn: func(t *testing.T, item *WorkItem) {
				if item.ParsedSpec != nil {
					t.Errorf("expected nil ParsedSpec for unknown kind, got %T", item.ParsedSpec)
				}
			},
		},
		{
			name: "empty jira spec (missing required key)",
			item: &WorkItem{
				TypeMeta: TypeMeta{Kind: KindJira},
				Spec:     json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "clears stale parsed spec",
			item: &WorkItem{
				TypeMeta:   TypeMeta{Kind: KindJira},
				Spec:       json.RawMessage(`{"key":"ARO-1"}`),
				ParsedSpec: &PRSpec{Number: 999},
			},
			checkFn: func(t *testing.T, item *WorkItem) {
				if _, ok := item.ParsedSpec.(*JiraSpec); !ok {
					t.Errorf("expected *JiraSpec after re-unmarshal, got %T", item.ParsedSpec)
				}
			},
		},
		{
			name: "malformed JSON",
			item: &WorkItem{
				TypeMeta: TypeMeta{Kind: KindJira},
				Spec:     json.RawMessage(`{broken`),
			},
			wantErr: true,
			checkFn: func(t *testing.T, item *WorkItem) {
				if item.ParsedSpec != nil {
					t.Errorf("expected nil ParsedSpec after failed unmarshal, got %T", item.ParsedSpec)
				}
			},
		},
		{
			name: "empty spec bytes",
			item: &WorkItem{
				TypeMeta: TypeMeta{Kind: KindJira},
			},
			checkFn: func(t *testing.T, item *WorkItem) {
				if item.ParsedSpec != nil {
					t.Errorf("expected nil ParsedSpec for empty spec, got %T", item.ParsedSpec)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.item.UnmarshalSpec()
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tt.item)
			}
		})
	}
}

func TestUnmarshalSpec_InvalidEnum(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind Kind
		spec string
	}{
		{
			name: "invalid staleness",
			kind: KindJira,
			spec: `{"key":"ARO-1","staleness":"garbage"}`,
		},
		{
			name: "invalid branch_state",
			kind: KindPR,
			spec: `{"repo":"org/repo","number":1,"branch":"main","branch_state":"garbage"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := &WorkItem{
				TypeMeta: TypeMeta{Kind: tt.kind},
				Spec:     json.RawMessage(tt.spec),
			}
			if err := item.UnmarshalSpec(); err == nil {
				t.Error("expected error for invalid enum")
			}
		})
	}
}

func TestUnmarshalSpec_RequiredFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind Kind
		spec string
	}{
		{
			name: "jira missing key",
			kind: KindJira,
			spec: `{}`,
		},
		{
			name: "jira invalid key format",
			kind: KindJira,
			spec: `{"key":"NODASH"}`,
		},
		{
			name: "pr missing repo",
			kind: KindPR,
			spec: `{"number":1,"branch":"main"}`,
		},
		{
			name: "pr missing number",
			kind: KindPR,
			spec: `{"repo":"org/repo","branch":"main"}`,
		},
		{
			name: "pr missing branch",
			kind: KindPR,
			spec: `{"repo":"org/repo","number":1}`,
		},
		{
			name: "pr invalid repo format",
			kind: KindPR,
			spec: `{"repo":"noslash","number":1,"branch":"main"}`,
		},
		{
			name: "pr negative number",
			kind: KindPR,
			spec: `{"repo":"org/repo","number":-1,"branch":"main"}`,
		},
		{
			name: "check missing name",
			kind: KindCheck,
			spec: `{}`,
		},
		{
			name: "review missing file",
			kind: KindReview,
			spec: `{}`,
		},
		{
			name: "local missing worktree_id",
			kind: KindLocal,
			spec: `{"branch":"main"}`,
		},
		{
			name: "local missing branch",
			kind: KindLocal,
			spec: `{"worktree_id":"abc"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := &WorkItem{
				TypeMeta: TypeMeta{Kind: tt.kind},
				Spec:     json.RawMessage(tt.spec),
			}
			if err := item.UnmarshalSpec(); err == nil {
				t.Error("expected validation error for missing required field")
			}
		})
	}
}

func TestUnmarshalSpecRecursive_NilChild(t *testing.T) {
	t.Parallel()
	item := &WorkItem{
		TypeMeta: TypeMeta{Kind: KindJira},
		Spec:     json.RawMessage(`{"key":"ARO-1"}`),
		Children: []*WorkItem{nil},
	}
	if err := item.UnmarshalSpecRecursive(); err == nil {
		t.Error("expected error for nil child")
	}
}

func TestNewWorkItem_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind Kind
		spec any
	}{
		{
			name: "invalid kind",
			kind: Kind("bogus"),
			spec: &JiraSpec{Key: "ARO-1"},
		},
		{
			name: "invalid spec (missing key)",
			kind: KindJira,
			spec: &JiraSpec{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewWorkItem(tt.kind, "id", "label", "status", tt.spec)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestMarshalSpec_Error(t *testing.T) {
	t.Parallel()
	_, err := MarshalSpec(make(chan int))
	if err == nil {
		t.Error("expected error for unmarshalable spec")
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		item *WorkItem
	}{
		{
			name: "jira with active staleness",
			item: func() *WorkItem {
				w, err := NewWorkItem(KindJira, "jira:ARO-99999", "Test issue", "New", &JiraSpec{
					Key:       "ARO-99999",
					Staleness: StalenessActive,
				})
				if err != nil {
					t.Fatal(err)
				}
				return w
			}(),
		},
		{
			name: "jira with zero staleness",
			item: func() *WorkItem {
				w, err := NewWorkItem(KindJira, "jira:ARO-1", "Test", "New", &JiraSpec{
					Key:       "ARO-1",
					Staleness: StalenessUnknown,
				})
				if err != nil {
					t.Fatal(err)
				}
				return w
			}(),
		},
		{
			name: "pr with zero branch state",
			item: func() *WorkItem {
				w, err := NewWorkItem(KindPR, "pr:org/repo:1", "Test PR", "open", &PRSpec{
					Repo:        "org/repo",
					Number:      1,
					Branch:      "main",
					BranchState: BranchStateUnknown,
				})
				if err != nil {
					t.Fatal(err)
				}
				return w
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
			if decoded.ID != tt.item.ID {
				t.Errorf("expected id %s, got %s", tt.item.ID, decoded.ID)
			}
		})
	}
}
