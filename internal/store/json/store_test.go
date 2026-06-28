package json

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/workitem"
)

func testItem(id, label, status string) *workitem.WorkItem {
	return &workitem.WorkItem{
		TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
		ObjectMeta: workitem.ObjectMeta{ID: id, Label: label, Status: status},
		Spec:       json.RawMessage(`{"key":"ARO-1"}`),
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
			items: []*workitem.WorkItem{testItem("jira:ARO-1", "Fix bug", "New")},
		},
		{
			name: "multiple items",
			items: []*workitem.WorkItem{
				testItem("jira:ARO-1", "Fix bug", "New"),
				testItem("jira:ARO-2", "Add feature", "Done"),
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
					TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
					ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Parent", Status: "New"},
					Spec:       json.RawMessage(`{"key":"ARO-1"}`),
					Children: []*workitem.WorkItem{
						{
							TypeMeta:   workitem.TypeMeta{Kind: workitem.KindPR},
							ObjectMeta: workitem.ObjectMeta{ID: "pr:1", Label: "PR", Status: "open"},
							Spec:       json.RawMessage(`{"repo":"Azure/ARO-HCP","number":1,"branch":"main"}`),
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
				if item.ID != tt.items[i].ID {
					t.Errorf("item[%d] ID = %s, want %s", i, item.ID, tt.items[i].ID)
				}
				if item.Label != tt.items[i].Label {
					t.Errorf("item[%d] Label = %s, want %s", i, item.Label, tt.items[i].Label)
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
		t.Fatalf("Load() should not return error for corrupt file, got: %v", err)
	}
	if items != nil {
		t.Fatalf("expected nil items for corrupt file, got %d", len(items))
	}
}

func TestStore_LoadDropsInvalidSpecs(t *testing.T) {
	t.Parallel()
	data := `[
		{"kind":"jira","id":"jira:ARO-1","label":"Good","status":"New","spec":{"key":"ARO-1"}},
		{"kind":"jira","id":"jira:ARO-2","label":"Bad","status":"New","spec":{"key":""}},
		{"kind":"jira","id":"jira:ARO-3","label":"Also good","status":"Done","spec":{"key":"ARO-3"}}
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
		t.Fatalf("expected 2 valid items (1 dropped), got %d", len(items))
	}
	if items[0].ID != "jira:ARO-1" {
		t.Errorf("first item ID = %s, want jira:ARO-1", items[0].ID)
	}
	if items[1].ID != "jira:ARO-3" {
		t.Errorf("second item ID = %s, want jira:ARO-3", items[1].ID)
	}
}

func TestStore_SaveAtomicNoPartialFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := New(path, logr.Discard())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	items := []*workitem.WorkItem{testItem("jira:ARO-1", "Test", "New")}
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

	if err := s.Save([]*workitem.WorkItem{testItem("jira:ARO-1", "First", "New")}); err != nil {
		t.Fatalf("first Save() error: %v", err)
	}
	if err := s.Save([]*workitem.WorkItem{testItem("jira:ARO-2", "Second", "Done")}); err != nil {
		t.Fatalf("second Save() error: %v", err)
	}

	items, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "jira:ARO-2" {
		t.Errorf("ID = %s, want jira:ARO-2", items[0].ID)
	}
}

func TestStore_LoadedItemsHaveParsedSpec(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := New(path, logr.Discard())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	items := []*workitem.WorkItem{testItem("jira:ARO-1", "Test", "New")}
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
