package informer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/geoberle/pulse/internal/workitem"
)

type expectedEvent struct {
	Type     EventType `json:"type"`
	OldID    string    `json:"old_id"`
	NewID    string    `json:"new_id"`
	ParentID string    `json:"parent_id"`
}

type diffFixture struct {
	Name           string             `json:"name"`
	Old            []*workitem.WorkItem `json:"old"`
	New            []*workitem.WorkItem `json:"new"`
	ExpectedEvents []expectedEvent     `json:"expected_events"`
}

func TestDiffTrees_GoldenFixtures(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test fixtures found")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}

			var fixture diffFixture
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatalf("unmarshal fixture %s: %v", file, err)
			}

			events := diffTrees(fixture.Old, fixture.New, nil)

			if len(events) != len(fixture.ExpectedEvents) {
				t.Fatalf("%s: expected %d events, got %d\nevents: %s",
					fixture.Name, len(fixture.ExpectedEvents), len(events), formatEvents(events))
			}

			for i, want := range fixture.ExpectedEvents {
				got := events[i]
				if got.Type != want.Type {
					t.Errorf("%s: event[%d] type = %s, want %s", fixture.Name, i, got.Type, want.Type)
				}
				gotOldID := ""
				if got.Old != nil {
					gotOldID = got.Old.ID
				}
				if gotOldID != want.OldID {
					t.Errorf("%s: event[%d] old_id = %q, want %q", fixture.Name, i, gotOldID, want.OldID)
				}
				gotNewID := ""
				if got.New != nil {
					gotNewID = got.New.ID
				}
				if gotNewID != want.NewID {
					t.Errorf("%s: event[%d] new_id = %q, want %q", fixture.Name, i, gotNewID, want.NewID)
				}
				gotParentID := ""
				if got.Parent != nil {
					gotParentID = got.Parent.ID
				}
				if gotParentID != want.ParentID {
					t.Errorf("%s: event[%d] parent_id = %q, want %q", fixture.Name, i, gotParentID, want.ParentID)
				}
			}
		})
	}
}

func TestDiffTrees_NilItemsSkipped(t *testing.T) {
	t.Parallel()
	oldItems := []*workitem.WorkItem{
		nil,
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
	}
	newItems := []*workitem.WorkItem{
		{
			TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
			ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
			Spec:       json.RawMessage(`{"key":"ARO-1"}`),
		},
		nil,
	}

	events := diffTrees(oldItems, newItems, nil)

	if len(events) != 0 {
		t.Fatalf("expected 0 events with nil items skipped, got %d: %s", len(events), formatEvents(events))
	}
}

func TestCanonicalizeJSON_KeyOrder(t *testing.T) {
	t.Parallel()
	a := json.RawMessage(`{"key":"ARO-1","staleness":"Active"}`)
	b := json.RawMessage(`{"staleness":"Active","key":"ARO-1"}`)

	if string(canonicalizeJSON(a)) != string(canonicalizeJSON(b)) {
		t.Error("expected identical canonical form for semantically equal JSON")
	}
}

func TestCanonicalizeJSON_InvalidFallback(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{broken`)
	result := canonicalizeJSON(raw)

	if string(result) != string(raw) {
		t.Errorf("expected fallback to raw bytes, got %s", string(result))
	}
}

func TestHashItem_CanonicalSpecComparison(t *testing.T) {
	t.Parallel()
	item1 := &workitem.WorkItem{
		TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
		ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
		Spec:       json.RawMessage(`{"key":"ARO-1","staleness":"Active"}`),
	}
	item2 := &workitem.WorkItem{
		TypeMeta:   workitem.TypeMeta{Kind: workitem.KindJira},
		ObjectMeta: workitem.ObjectMeta{ID: "jira:ARO-1", Label: "Test", Status: "New"},
		Spec:       json.RawMessage(`{"staleness":"Active","key":"ARO-1"}`),
	}

	if hashItem(item1) != hashItem(item2) {
		t.Error("expected same hash for semantically equal specs with different key order")
	}
}

func formatEvents(events []Event) string {
	data, _ := json.MarshalIndent(events, "", "  ")
	return string(data)
}
