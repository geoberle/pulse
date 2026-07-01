package json

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/geoberle/pulse/internal/workitem"
)

func testItem(name string, phase workitem.WorkItemPhase) *workitem.WorkItem {
	return &workitem.WorkItem{
		TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindJira)},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Test"}`),
		Status:     workitem.WorkItemStatus{Phase: phase},
		ParsedSpec: &workitem.JiraSpec{Key: "ARO-1", Summary: "Test"},
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		items []*workitem.WorkItem
	}{
		{
			name:  "single item",
			items: []*workitem.WorkItem{testItem("jira.aro-1", "New")},
		},
		{
			name: "multiple items",
			items: []*workitem.WorkItem{
				testItem("jira.aro-1", "New"),
				testItem("jira.aro-2", "Done"),
			},
		},
		{
			name:  "empty list",
			items: []*workitem.WorkItem{},
		},
		{
			name: "item with children",
			items: []*workitem.WorkItem{
				{
					TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindJira)},
					ObjectMeta: metav1.ObjectMeta{Name: "jira.aro-1"},
					Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Parent"}`),
					Status:     workitem.WorkItemStatus{Phase: "New"},
					ParsedSpec: &workitem.JiraSpec{Key: "ARO-1", Summary: "Parent"},
					Children: []*workitem.WorkItem{
						{
							TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindPR)},
							ObjectMeta: metav1.ObjectMeta{Name: "pr.1"},
							Spec:       json.RawMessage(`{"repo":"Azure/ARO-HCP","number":1,"branch":"main","title":"PR"}`),
							Status:     workitem.WorkItemStatus{Phase: "open"},
							ParsedSpec: &workitem.PRSpec{Repo: "Azure/ARO-HCP", Number: 1, Branch: "main", Title: "PR"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "state.json")
			s, err := New(path, logr.Discard())
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			if err := s.Save(tt.items); err != nil {
				t.Fatalf("Save() error: %v", err)
			}

			loaded, err := s.Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}

			if len(loaded) != len(tt.items) {
				t.Fatalf("expected %d items, got %d", len(tt.items), len(loaded))
			}

			for i, item := range loaded {
				if item.Name != tt.items[i].Name {
					t.Errorf("item[%d] Name = %s, want %s", i, item.Name, tt.items[i].Name)
				}
				if item.DisplayName() != tt.items[i].DisplayName() {
					t.Errorf("item[%d] DisplayName = %s, want %s", i, item.DisplayName(), tt.items[i].DisplayName())
				}
				if item.Kind != tt.items[i].Kind {
					t.Errorf("item[%d] Kind = %s, want %s", i, item.Kind, tt.items[i].Kind)
				}
			}
		})
	}
}

func TestStore_LoadMissingFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nonexistent", "state.json")
	s := &Store{path: path, log: logr.Discard()}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if items != nil {
		t.Fatalf("expected nil items for missing file, got %d", len(items))
	}
}

func TestStore_LoadCorruptFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("not json{{{"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{path: path, log: logr.Discard()}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() should tolerate corrupt file, got error: %v", err)
	}
	if items != nil {
		t.Fatalf("expected nil items for corrupt file, got %d", len(items))
	}
}

func TestStore_LoadToleratesInvalidSpecs(t *testing.T) {
	t.Parallel()
	data := `[
		{"kind":"jira","metadata":{"name":"jira:ARO-1"},"status":{"phase":"New"},"spec":{"key":"ARO-1","summary":"Good"}},
		{"kind":"jira","metadata":{"name":"jira:ARO-2"},"status":{"phase":"New"},"spec":{"key":"","summary":"Bad key"}},
		{"kind":"jira","metadata":{"name":"jira:ARO-3"},"status":{"phase":"Done"},"spec":{"key":"ARO-3","summary":"Also good"}}
	]`
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{path: path, log: logr.Discard()}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items (tolerate invalid specs on read), got %d", len(items))
	}
}

func TestStore_LoadKeepsUnknownKind(t *testing.T) {
	t.Parallel()
	data := `[
		{"kind":"agent","metadata":{"name":"agent:task-1"},"status":{"phase":"running"},"spec":{"task":"do stuff"}},
		{"kind":"jira","metadata":{"name":"jira:ARO-1"},"status":{"phase":"New"},"spec":{"key":"ARO-1","summary":"Known"}}
	]`
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{path: path, log: logr.Discard()}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items (unknown kind kept), got %d", len(items))
	}
	if items[0].Name != "agent.task-1" {
		t.Errorf("first item Name = %s, want agent.task-1", items[0].Name)
	}
	if items[0].ParsedSpec != nil {
		t.Errorf("unknown kind should have nil ParsedSpec, got %T", items[0].ParsedSpec)
	}
	if items[0].DisplayName() != "agent.task-1" {
		t.Errorf("DisplayName = %s, want fallback to Name", items[0].DisplayName())
	}
	if items[1].ParsedSpec == nil {
		t.Error("known kind should have non-nil ParsedSpec")
	}
}

func TestStore_SaveAtomicNoPartialFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := New(path, logr.Discard())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	items := []*workitem.WorkItem{testItem("jira.aro-1", "New")}
	if err := s.Save(items); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify no temp files left behind
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "state.json" {
			t.Errorf("unexpected file in state dir: %s", e.Name())
		}
	}
}

func TestStore_NewCreatesDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "deep", "state.json")
	_, err := New(path, logr.Discard())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory")
	}
}

func TestStore_SaveOverwritesPrevious(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := New(path, logr.Discard())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := s.Save([]*workitem.WorkItem{testItem("jira.aro-1", "New")}); err != nil {
		t.Fatalf("first Save() error: %v", err)
	}
	if err := s.Save([]*workitem.WorkItem{testItem("jira.aro-2", "Done")}); err != nil {
		t.Fatalf("second Save() error: %v", err)
	}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "jira.aro-2" {
		t.Errorf("Name = %s, want jira.aro-2", items[0].Name)
	}
}

func TestStore_SaveLoadPreservesTreeShape(t *testing.T) {
	t.Parallel()
	tree := []*workitem.WorkItem{
		{
			TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindJira)},
			ObjectMeta: metav1.ObjectMeta{Name: "jira.aro-1"},
			Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Root"}`),
			Status:     workitem.WorkItemStatus{Phase: "New"},
			ParsedSpec: &workitem.JiraSpec{Key: "ARO-1", Summary: "Root"},
			Children: []*workitem.WorkItem{
				{
					TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindPR)},
					ObjectMeta: metav1.ObjectMeta{Name: "pr.repo.1"},
					Spec:       json.RawMessage(`{"repo":"Azure/ARO-HCP","number":1,"branch":"main","title":"PR 1"}`),
					Status:     workitem.WorkItemStatus{Phase: "open"},
					ParsedSpec: &workitem.PRSpec{Repo: "Azure/ARO-HCP", Number: 1, Branch: "main", Title: "PR 1"},
					Children: []*workitem.WorkItem{
						{
							TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindCheck)},
							ObjectMeta: metav1.ObjectMeta{Name: "check.ci"},
							Spec:       json.RawMessage(`{"name":"ci"}`),
							Status:     workitem.WorkItemStatus{Phase: "pass"},
							ParsedSpec: &workitem.CheckSpec{Name: "ci"},
						},
						{
							TypeMeta:   metav1.TypeMeta{Kind: string(workitem.KindReview)},
							ObjectMeta: metav1.ObjectMeta{Name: "review.1"},
							Spec:       json.RawMessage(`{"file":"main.go"}`),
							Status:     workitem.WorkItemStatus{Phase: "pending"},
							ParsedSpec: &workitem.ReviewSpec{File: "main.go"},
						},
					},
				},
			},
		},
	}

	path := filepath.Join(t.TempDir(), "state.json")
	s, err := New(path, logr.Discard())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := s.Save(tree); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 root, got %d", len(loaded))
	}
	root := loaded[0]
	if root.Name != "jira.aro-1" {
		t.Errorf("root Name = %s, want jira.aro-1", root.Name)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 PR child, got %d", len(root.Children))
	}
	pr := root.Children[0]
	if pr.Name != "pr.repo.1" {
		t.Errorf("PR Name = %s, want pr.repo.1", pr.Name)
	}
	if len(pr.Children) != 2 {
		t.Fatalf("expected 2 PR children (check+review), got %d", len(pr.Children))
	}
	if pr.Children[0].Name != "check.ci" {
		t.Errorf("check Name = %s, want check.ci", pr.Children[0].Name)
	}
	if pr.Children[1].Name != "review.1" {
		t.Errorf("review Name = %s, want review.1", pr.Children[1].Name)
	}
}

func TestStore_NewFailsOnBadDirectory(t *testing.T) {
	t.Parallel()
	// Use a file as parent so MkdirAll fails
	tmpFile := filepath.Join(t.TempDir(), "blockfile")
	if err := os.WriteFile(tmpFile, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(tmpFile, "nested", "state.json")
	_, err := New(path, logr.Discard())
	if err == nil {
		t.Fatal("expected error when parent is a file")
	}
}

func TestStore_LoadedItemsHaveParsedSpec(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := New(path, logr.Discard())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	items := []*workitem.WorkItem{testItem("jira.aro-1", "New")}
	if err := s.Save(items); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 item, got %d", len(loaded))
	}
	if loaded[0].ParsedSpec == nil {
		t.Fatal("expected ParsedSpec to be populated after Load")
	}
	jiraSpec, ok := loaded[0].ParsedSpec.(*workitem.JiraSpec)
	if !ok {
		t.Fatalf("expected *JiraSpec, got %T", loaded[0].ParsedSpec)
	}
	if jiraSpec.Key != "ARO-1" {
		t.Errorf("Key = %s, want ARO-1", jiraSpec.Key)
	}
}

func TestStore_LoadNormalizesOldNames(t *testing.T) {
	t.Parallel()
	data := `[
		{
			"kind":"jira",
			"metadata":{"name":"jira:ARO-123"},
			"status":{"phase":"New"},
			"spec":{"key":"ARO-123","summary":"Old format"},
			"children":[
				{
					"kind":"pr",
					"metadata":{"name":"pr:Azure/ARO-HCP:891"},
					"status":{"phase":"open"},
					"spec":{"repo":"Azure/ARO-HCP","number":891,"branch":"main","title":"PR"},
					"children":[
						{
							"kind":"check",
							"metadata":{"name":"check:99"},
							"status":{"phase":"pass"},
							"spec":{"name":"ci"}
						}
					]
				}
			]
		}
	]`
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{path: path, log: logr.Discard()}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 root, got %d", len(items))
	}

	tests := []struct {
		name string
		want string
		got  string
	}{
		{"root", "jira.aro-123", items[0].Name},
		{"child", "pr.azure.aro-hcp.891", items[0].Children[0].Name},
		{"grandchild", "check.99", items[0].Children[0].Children[0].Name},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: Name = %s, want %s", tt.name, tt.got, tt.want)
		}
	}
}

func TestStore_LoadNormalizesIdempotent(t *testing.T) {
	t.Parallel()
	data := `[
		{"kind":"jira","metadata":{"name":"jira.aro-123"},"status":{"phase":"New"},"spec":{"key":"ARO-123","summary":"Already normalized"}}
	]`
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{path: path, log: logr.Discard()}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if items[0].Name != "jira.aro-123" {
		t.Errorf("Name = %s, want jira.aro-123", items[0].Name)
	}
}
