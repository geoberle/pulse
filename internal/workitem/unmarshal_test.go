package workitem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if jiraSpec.Summary != "Implement DNS migration" {
		t.Errorf("expected summary %q, got %q", "Implement DNS migration", jiraSpec.Summary)
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
	if prSpec.Title != "PR #891 (ARO-HCP)" {
		t.Errorf("expected title %q, got %q", "PR #891 (ARO-HCP)", prSpec.Title)
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
				if spec.Title != "fix typo in README" {
					t.Errorf("expected title %q, got %q", "fix typo in README", spec.Title)
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
				TypeMeta: metav1.TypeMeta{Kind: "unknown"},
				Spec:     json.RawMessage(`{}`),
			},
			checkFn: func(t *testing.T, item *WorkItem) {
				if item.ParsedSpec != nil {
					t.Errorf("expected nil ParsedSpec for unknown kind, got %T", item.ParsedSpec)
				}
			},
		},
		{
			name: "empty jira spec (tolerates missing fields on read)",
			item: &WorkItem{
				TypeMeta: metav1.TypeMeta{Kind: string(KindJira)},
				Spec:     json.RawMessage(`{}`),
			},
			checkFn: func(t *testing.T, item *WorkItem) {
				spec, ok := item.ParsedSpec.(*JiraSpec)
				if !ok {
					t.Fatalf("expected *JiraSpec, got %T", item.ParsedSpec)
				}
				if len(spec.Key) != 0 {
					t.Errorf("expected empty Key, got %q", spec.Key)
				}
			},
		},
		{
			name: "jira spec with missing summary (tolerates on read)",
			item: &WorkItem{
				TypeMeta: metav1.TypeMeta{Kind: string(KindJira)},
				Spec:     json.RawMessage(`{"key":"ARO-1"}`),
			},
			checkFn: func(t *testing.T, item *WorkItem) {
				spec, ok := item.ParsedSpec.(*JiraSpec)
				if !ok {
					t.Fatalf("expected *JiraSpec, got %T", item.ParsedSpec)
				}
				if spec.Key != "ARO-1" {
					t.Errorf("expected Key ARO-1, got %q", spec.Key)
				}
			},
		},
		{
			name: "clears stale parsed spec",
			item: &WorkItem{
				TypeMeta:   metav1.TypeMeta{Kind: string(KindJira)},
				Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Test"}`),
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
				TypeMeta: metav1.TypeMeta{Kind: string(KindJira)},
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
				TypeMeta: metav1.TypeMeta{Kind: string(KindJira)},
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

func TestUnmarshalSpec_ToleratesInvalidFields(t *testing.T) {
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
			spec: `{"key":"aro-1","summary":"test"}`,
		},
		{
			name: "pr missing repo",
			kind: KindPR,
			spec: `{"number":1,"branch":"main","title":"t"}`,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := &WorkItem{
				TypeMeta: metav1.TypeMeta{Kind: string(tt.kind)},
				Spec:     json.RawMessage(tt.spec),
			}
			if err := item.UnmarshalSpec(); err != nil {
				t.Errorf("UnmarshalSpec() should tolerate invalid fields on read, got error: %v", err)
			}
			if item.ParsedSpec == nil {
				t.Error("expected ParsedSpec to be set even with invalid fields")
			}
		})
	}
}

func TestNewWorkItem_RejectsInvalidSpecs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind Kind
		spec Spec
	}{
		{name: "jira missing key", kind: KindJira, spec: &JiraSpec{}},
		{name: "jira missing summary", kind: KindJira, spec: &JiraSpec{Key: "ARO-1"}},
		{name: "jira invalid key format", kind: KindJira, spec: &JiraSpec{Key: "aro-1", Summary: "test"}},
		{name: "pr missing repo", kind: KindPR, spec: &PRSpec{Number: 1, Branch: "main", Title: "t"}},
		{name: "pr invalid repo format", kind: KindPR, spec: &PRSpec{Repo: "noslash", Number: 1, Branch: "main", Title: "t"}},
		{name: "pr negative number", kind: KindPR, spec: &PRSpec{Repo: "org/repo", Number: -1, Branch: "main", Title: "t"}},
		{name: "check missing name", kind: KindCheck, spec: &CheckSpec{}},
		{name: "review missing file", kind: KindReview, spec: &ReviewSpec{}},
		{name: "local missing worktree_id", kind: KindLocal, spec: &LocalSpec{Branch: "main"}},
		{name: "local missing branch", kind: KindLocal, spec: &LocalSpec{WorktreeID: "abc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewWorkItem(tt.kind, "test-item", "phase", tt.spec)
			if err == nil {
				t.Error("expected NewWorkItem to reject invalid spec")
			}
		})
	}
}

func TestUnmarshalSpecRecursive_NilChild(t *testing.T) {
	t.Parallel()
	item := &WorkItem{
		TypeMeta: metav1.TypeMeta{Kind: string(KindJira)},
		Spec:     json.RawMessage(`{"key":"ARO-1","summary":"Test"}`),
		Children: []*WorkItem{nil},
	}
	if err := item.UnmarshalSpecRecursive(); err == nil {
		t.Error("expected error for nil child")
	}
}

func TestNewWorkItem_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		kind  Kind
		id    string
		phase WorkItemPhase
		spec  Spec
	}{
		{
			name:  "invalid kind",
			kind:  Kind("bogus"),
			id:    "id",
			phase: "status",
			spec:  &JiraSpec{Key: "ARO-1", Summary: "Test"},
		},
		{
			name:  "invalid spec (missing key)",
			kind:  KindJira,
			id:    "id",
			phase: "status",
			spec:  &JiraSpec{},
		},
		{
			name:  "id too long",
			kind:  KindJira,
			id:    strings.Repeat("a", 501),
			phase: "status",
			spec:  &JiraSpec{Key: "ARO-1", Summary: "Test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewWorkItem(tt.kind, tt.id, tt.phase, tt.spec)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

type unmarshalableSpec struct{}

func (unmarshalableSpec) Validate() error      { return nil }
func (s unmarshalableSpec) DeepCopySpec() Spec { return s }
func (unmarshalableSpec) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("forced marshal error")
}

func TestMarshalSpec_Error(t *testing.T) {
	t.Parallel()
	_, err := MarshalSpec(unmarshalableSpec{})
	if err == nil {
		t.Error("expected error for unmarshalable spec")
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		item    *WorkItem
		checkFn func(t *testing.T, decoded *WorkItem)
	}{
		{
			name: "jira roundtrip",
			item: func() *WorkItem {
				w, err := NewWorkItem(KindJira, "jira:ARO-99999", "New", &JiraSpec{
					Key:     "ARO-99999",
					Summary: "Test issue",
				})
				if err != nil {
					t.Fatal(err)
				}
				return w
			}(),
			checkFn: func(t *testing.T, decoded *WorkItem) {
				spec := decoded.ParsedSpec.(*JiraSpec)
				if spec.Key != "ARO-99999" {
					t.Errorf("expected key %q, got %q", "ARO-99999", spec.Key)
				}
				if spec.Summary != "Test issue" {
					t.Errorf("expected summary %q, got %q", "Test issue", spec.Summary)
				}
			},
		},
		{
			name: "pr roundtrip",
			item: func() *WorkItem {
				w, err := NewWorkItem(KindPR, "pr:org/repo:1", "open", &PRSpec{
					Repo:   "org/repo",
					Number: 1,
					Branch: "main",
					Title:  "Test PR",
				})
				if err != nil {
					t.Fatal(err)
				}
				return w
			}(),
			checkFn: func(t *testing.T, decoded *WorkItem) {
				spec := decoded.ParsedSpec.(*PRSpec)
				if spec.Repo != "org/repo" {
					t.Errorf("expected repo %q, got %q", "org/repo", spec.Repo)
				}
				if spec.Title != "Test PR" {
					t.Errorf("expected title %q, got %q", "Test PR", spec.Title)
				}
			},
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
			if decoded.Name != tt.item.Name {
				t.Errorf("expected name %s, got %s", tt.item.Name, decoded.Name)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, &decoded)
			}
		})
	}
}

func TestDisplayName_AllKinds(t *testing.T) {
	t.Parallel()
	for kind := range specFactories {
		item := MakeTestItem(kind, "test.1")
		got := item.DisplayName()
		if got == item.Name {
			t.Errorf("kind %s: DisplayName returned raw Name, missing switch case", kind)
		}
	}
}
