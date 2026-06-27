package engine

import (
	"testing"

	"github.com/geoberle/pulse/internal/workitem"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name      string
		items     []*workitem.WorkItem
		wantCount int
	}{
		{
			name:      "nil input returns nil",
			items:     nil,
			wantCount: 0,
		},
		{
			name:      "empty input returns empty",
			items:     []*workitem.WorkItem{},
			wantCount: 0,
		},
		{
			name: "passthrough preserves all items",
			items: []*workitem.WorkItem{
				makeItem(workitem.KindPR, "pr:test/repo:1", "first"),
				makeItem(workitem.KindPR, "pr:test/repo:2", "second"),
			},
			wantCount: 2,
		},
		{
			name: "preserves children",
			items: func() []*workitem.WorkItem {
				pr := makeItem(workitem.KindPR, "pr:test/repo:1", "with children")
				pr.Children = []*workitem.WorkItem{
					makeItem(workitem.KindCheck, "check:100", "lint"),
				}
				return []*workitem.WorkItem{pr}
			}(),
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merge(tt.items)
			if len(result) != tt.wantCount {
				t.Fatalf("merge returned %d items, want %d", len(result), tt.wantCount)
			}
			for i, item := range result {
				if item.ID != tt.items[i].ID {
					t.Errorf("item[%d].ID = %q, want %q", i, item.ID, tt.items[i].ID)
				}
			}
		})
	}
}
