package workitem

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFlatten_SingleRoot(t *testing.T) {
	t.Parallel()
	root := &WorkItem{
		TypeMeta:   metav1.TypeMeta{Kind: string(KindJira)},
		ObjectMeta: metav1.ObjectMeta{Name: "jira:ARO-1"},
		Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Root"}`),
		Status:     WorkItemStatus{Phase: "New"},
	}

	flat := Flatten([]*WorkItem{root})

	if len(flat) != 1 {
		t.Fatalf("expected 1 item, got %d", len(flat))
	}
	if flat[0].Name != "jira:ARO-1" {
		t.Errorf("expected Name jira:ARO-1, got %s", flat[0].Name)
	}
	if flat[0].ParentName() != "" {
		t.Errorf("expected empty ParentName, got %q", flat[0].ParentName())
	}
}

// TestFlatten_TreeSetsParentID verifies depth-first pre-order: parents before
// children. TUI display relies on this ordering.
func TestFlatten_TreeSetsParentID(t *testing.T) {
	t.Parallel()
	root := &WorkItem{
		TypeMeta:   metav1.TypeMeta{Kind: string(KindJira)},
		ObjectMeta: metav1.ObjectMeta{Name: "jira:ARO-1"},
		Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Root"}`),
		Status:     WorkItemStatus{Phase: "New"},
		Children: []*WorkItem{
			{
				TypeMeta:   metav1.TypeMeta{Kind: string(KindPR)},
				ObjectMeta: metav1.ObjectMeta{Name: "pr:repo:1"},
				Spec:       json.RawMessage(`{"repo":"test/repo","number":1,"branch":"main","title":"PR 1"}`),
				Status:     WorkItemStatus{Phase: "open"},
				Children: []*WorkItem{
					{
						TypeMeta:   metav1.TypeMeta{Kind: string(KindCheck)},
						ObjectMeta: metav1.ObjectMeta{Name: "check:ci"},
						Spec:       json.RawMessage(`{"name":"ci"}`),
						Status:     WorkItemStatus{Phase: "pass"},
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
		name       string
		parentName string
	}{
		{"jira:ARO-1", ""},
		{"pr:repo:1", "jira:ARO-1"},
		{"check:ci", "pr:repo:1"},
	}
	for i, tt := range tests {
		if flat[i].Name != tt.name {
			t.Errorf("item %d: expected Name %s, got %s", i, tt.name, flat[i].Name)
		}
		if flat[i].ParentName() != tt.parentName {
			t.Errorf("item %d: expected ParentName %q, got %q", i, tt.parentName, flat[i].ParentName())
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
		TypeMeta:   metav1.TypeMeta{Kind: string(KindJira)},
		ObjectMeta: metav1.ObjectMeta{Name: "jira:ARO-1"},
		Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Root"}`),
		Status:     WorkItemStatus{Phase: "New"},
		Children: []*WorkItem{
			{
				TypeMeta:   metav1.TypeMeta{Kind: string(KindPR)},
				ObjectMeta: metav1.ObjectMeta{Name: "pr:repo:1"},
				Spec:       json.RawMessage(`{"repo":"test/repo","number":1,"branch":"main","title":"PR 1"}`),
				Status:     WorkItemStatus{Phase: "open"},
			},
			{
				TypeMeta:   metav1.TypeMeta{Kind: string(KindPR)},
				ObjectMeta: metav1.ObjectMeta{Name: "pr:repo:2"},
				Spec:       json.RawMessage(`{"repo":"test/repo","number":2,"branch":"main","title":"PR 2"}`),
				Status:     WorkItemStatus{Phase: "open"},
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
	if tree[0].Name != "jira:ARO-1" {
		t.Errorf("expected root Name jira:ARO-1, got %s", tree[0].Name)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].Name != "pr:repo:1" {
		t.Errorf("expected child 0 Name pr:repo:1, got %s", tree[0].Children[0].Name)
	}
	if tree[0].Children[1].Name != "pr:repo:2" {
		t.Errorf("expected child 1 Name pr:repo:2, got %s", tree[0].Children[1].Name)
	}
	if tree[0].ParentName() != "" {
		t.Errorf("expected empty ParentName on root after BuildTree, got %q", tree[0].ParentName())
	}
}

func TestBuildTree_OrphanedChild(t *testing.T) {
	t.Parallel()
	items := []*WorkItem{
		{
			TypeMeta:   metav1.TypeMeta{Kind: string(KindPR)},
			ObjectMeta: metav1.ObjectMeta{Name: "pr:1", OwnerReferences: []metav1.OwnerReference{{Name: "jira:missing"}}},
			Status:     WorkItemStatus{Phase: "open"},
		},
	}

	tree := BuildTree(items)

	if len(tree) != 1 {
		t.Fatalf("expected orphan to become root, got %d roots", len(tree))
	}
	if tree[0].Name != "pr:1" {
		t.Errorf("expected pr:1, got %s", tree[0].Name)
	}
}

func TestFlatten_SiblingOrder(t *testing.T) {
	t.Parallel()
	root := &WorkItem{
		TypeMeta:   metav1.TypeMeta{Kind: string(KindJira)},
		ObjectMeta: metav1.ObjectMeta{Name: "jira:ARO-1"},
		Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Root"}`),
		Status:     WorkItemStatus{Phase: "New"},
		Children: []*WorkItem{
			{
				TypeMeta:   metav1.TypeMeta{Kind: string(KindPR)},
				ObjectMeta: metav1.ObjectMeta{Name: "pr:repo:1"},
				Spec:       json.RawMessage(`{"repo":"test/repo","number":1,"branch":"main","title":"PR 1"}`),
				Status:     WorkItemStatus{Phase: "open"},
				Children: []*WorkItem{
					{
						TypeMeta:   metav1.TypeMeta{Kind: string(KindCheck)},
						ObjectMeta: metav1.ObjectMeta{Name: "check:a"},
						Spec:       json.RawMessage(`{"name":"ci-a"}`),
						Status:     WorkItemStatus{Phase: "pass"},
					},
				},
			},
			{
				TypeMeta:   metav1.TypeMeta{Kind: string(KindPR)},
				ObjectMeta: metav1.ObjectMeta{Name: "pr:repo:2"},
				Spec:       json.RawMessage(`{"repo":"test/repo","number":2,"branch":"main","title":"PR 2"}`),
				Status:     WorkItemStatus{Phase: "open"},
				Children: []*WorkItem{
					{
						TypeMeta:   metav1.TypeMeta{Kind: string(KindCheck)},
						ObjectMeta: metav1.ObjectMeta{Name: "check:b"},
						Spec:       json.RawMessage(`{"name":"ci-b"}`),
						Status:     WorkItemStatus{Phase: "fail"},
					},
				},
			},
		},
	}

	flat := Flatten([]*WorkItem{root})

	wantOrder := []string{"jira:ARO-1", "pr:repo:1", "check:a", "pr:repo:2", "check:b"}
	if len(flat) != len(wantOrder) {
		t.Fatalf("expected %d items, got %d", len(wantOrder), len(flat))
	}
	for i, want := range wantOrder {
		if flat[i].Name != want {
			t.Errorf("item %d: expected %s, got %s", i, want, flat[i].Name)
		}
	}
}

func TestBuildTree_Roundtrip_Depth3(t *testing.T) {
	t.Parallel()
	root := &WorkItem{
		TypeMeta:   metav1.TypeMeta{Kind: string(KindJira)},
		ObjectMeta: metav1.ObjectMeta{Name: "jira:ARO-1"},
		Spec:       json.RawMessage(`{"key":"ARO-1","summary":"Root"}`),
		Status:     WorkItemStatus{Phase: "New"},
		Children: []*WorkItem{
			{
				TypeMeta:   metav1.TypeMeta{Kind: string(KindPR)},
				ObjectMeta: metav1.ObjectMeta{Name: "pr:repo:1"},
				Spec:       json.RawMessage(`{"repo":"test/repo","number":1,"branch":"main","title":"PR 1"}`),
				Status:     WorkItemStatus{Phase: "open"},
				Children: []*WorkItem{
					{
						TypeMeta:   metav1.TypeMeta{Kind: string(KindCheck)},
						ObjectMeta: metav1.ObjectMeta{Name: "check:ci"},
						Spec:       json.RawMessage(`{"name":"ci"}`),
						Status:     WorkItemStatus{Phase: "pass"},
					},
					{
						TypeMeta:   metav1.TypeMeta{Kind: string(KindReview)},
						ObjectMeta: metav1.ObjectMeta{Name: "review:1"},
						Spec:       json.RawMessage(`{"file":"main.go"}`),
						Status:     WorkItemStatus{Phase: "pending"},
					},
				},
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
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 PR child, got %d", len(tree[0].Children))
	}
	pr := tree[0].Children[0]
	if pr.Name != "pr:repo:1" {
		t.Errorf("expected pr:repo:1, got %s", pr.Name)
	}
	if len(pr.Children) != 2 {
		t.Fatalf("expected 2 PR children (check+review), got %d", len(pr.Children))
	}
	if pr.Children[0].Name != "check:ci" {
		t.Errorf("expected check:ci, got %s", pr.Children[0].Name)
	}
	if pr.Children[1].Name != "review:1" {
		t.Errorf("expected review:1, got %s", pr.Children[1].Name)
	}
}

func TestBuildTree_Empty(t *testing.T) {
	t.Parallel()
	tree := BuildTree(nil)
	if len(tree) != 0 {
		t.Fatalf("expected 0 roots, got %d", len(tree))
	}
}
