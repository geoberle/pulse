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

func formatEvents(events []Event) string {
	data, _ := json.MarshalIndent(events, "", "  ")
	return string(data)
}
