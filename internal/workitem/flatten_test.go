package workitem

import (
	"encoding/json"
	"testing"
)

func TestFlatten_SingleRoot(t *testing.T) {
	t.Parallel()
	root := &WorkItem{
		TypeMeta:   TypeMeta{Kind: KindJira},
		ObjectMeta: ObjectMeta{ID: "jira:ARO-1", Label: "Root", Status: "New"},
		Spec:       json.RawMessage(`{"key":"ARO-1"}`),
	}

	flat := Flatten([]*WorkItem{root})

	if len(flat) != 1 {
		t.Fatalf("expected 1 item, got %d", len(flat))
	}
	if flat[0].ID != "jira:ARO-1" {
		t.Errorf("expected ID jira:ARO-1, got %s", flat[0].ID)
	}
	if flat[0].ParentID != "" {
		t.Errorf("expected empty ParentID, got %q", flat[0].ParentID)
	}
}

func TestFlatten_TreeSetsParentID(t *testing.T) {
	t.Parallel()
	root := &WorkItem{
		TypeMeta:   TypeMeta{Kind: KindJira},
		ObjectMeta: ObjectMeta{ID: "jira:ARO-1", Label: "Root", Status: "New"},
		Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		Children: []*WorkItem{
			{
				TypeMeta:   TypeMeta{Kind: KindPR},
				ObjectMeta: ObjectMeta{ID: "pr:repo:1", Label: "PR 1", Status: "open"},
				Spec:       json.RawMessage(`{"repo":"test","number":1}`),
				Children: []*WorkItem{
					{
						TypeMeta:   TypeMeta{Kind: KindCheck},
						ObjectMeta: ObjectMeta{ID: "check:ci", Label: "CI", Status: "pass"},
						Spec:       json.RawMessage(`{"name":"ci"}`),
					},
				},
			},
		},
	}

	flat := Flatten([]*WorkItem{root})

	if len(flat) != 3 {
		t.Fatalf("expected 3 items, got %d", len(flat))
	}

	tests := []struct {
		id       string
		parentID string
	}{
		{"jira:ARO-1", ""},
		{"pr:repo:1", "jira:ARO-1"},
		{"check:ci", "pr:repo:1"},
	}
	for i, tt := range tests {
		if flat[i].ID != tt.id {
			t.Errorf("item %d: expected ID %s, got %s", i, tt.id, flat[i].ID)
		}
		if flat[i].ParentID != tt.parentID {
			t.Errorf("item %d: expected ParentID %q, got %q", i, tt.parentID, flat[i].ParentID)
		}
		if len(flat[i].Children) != 0 {
			t.Errorf("item %d: expected no Children, got %d", i, len(flat[i].Children))
		}
	}
}

func TestFlatten_Empty(t *testing.T) {
	t.Parallel()
	flat := Flatten(nil)
	if len(flat) != 0 {
		t.Fatalf("expected 0 items, got %d", len(flat))
	}
}

func TestBuildTree_Roundtrip(t *testing.T) {
	t.Parallel()
	root := &WorkItem{
		TypeMeta:   TypeMeta{Kind: KindJira},
		ObjectMeta: ObjectMeta{ID: "jira:ARO-1", Label: "Root", Status: "New"},
		Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		Children: []*WorkItem{
			{
				TypeMeta:   TypeMeta{Kind: KindPR},
				ObjectMeta: ObjectMeta{ID: "pr:repo:1", Label: "PR 1", Status: "open"},
				Spec:       json.RawMessage(`{"repo":"test","number":1}`),
			},
			{
				TypeMeta:   TypeMeta{Kind: KindPR},
				ObjectMeta: ObjectMeta{ID: "pr:repo:2", Label: "PR 2", Status: "open"},
				Spec:       json.RawMessage(`{"repo":"test","number":2}`),
			},
		},
	}

	flat := Flatten([]*WorkItem{root})
	ptrs := make([]*WorkItem, len(flat))
	for i := range flat {
		ptrs[i] = &flat[i]
	}
	tree := BuildTree(ptrs)

	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}
	if tree[0].ID != "jira:ARO-1" {
		t.Errorf("expected root ID jira:ARO-1, got %s", tree[0].ID)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].ID != "pr:repo:1" {
		t.Errorf("expected child 0 ID pr:repo:1, got %s", tree[0].Children[0].ID)
	}
	if tree[0].Children[1].ID != "pr:repo:2" {
		t.Errorf("expected child 1 ID pr:repo:2, got %s", tree[0].Children[1].ID)
	}
	if tree[0].ParentID != "" {
		t.Errorf("expected empty ParentID on root after BuildTree, got %q", tree[0].ParentID)
	}
}

func TestBuildTree_OrphanedChild(t *testing.T) {
	t.Parallel()
	items := []*WorkItem{
		{
			TypeMeta:   TypeMeta{Kind: KindPR},
			ObjectMeta: ObjectMeta{ID: "pr:1", Label: "PR", Status: "open", ParentID: "jira:missing"},
		},
	}

	tree := BuildTree(items)

	if len(tree) != 1 {
		t.Fatalf("expected orphan to become root, got %d roots", len(tree))
	}
	if tree[0].ID != "pr:1" {
		t.Errorf("expected pr:1, got %s", tree[0].ID)
	}
}

func TestBuildTree_Empty(t *testing.T) {
	t.Parallel()
	tree := BuildTree(nil)
	if len(tree) != 0 {
		t.Fatalf("expected 0 roots, got %d", len(tree))
	}
}
