package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/poller"
	"github.com/geoberle/pulse/internal/workitem"
)

type mockPoller struct {
	pollFn func(ctx context.Context) ([]*workitem.WorkItem, error)
}

func (m *mockPoller) Poll(ctx context.Context) ([]*workitem.WorkItem, error) {
	return m.pollFn(ctx)
}

func mockPollerFn(fn func(ctx context.Context) ([]*workitem.WorkItem, error)) poller.Poller {
	return &mockPoller{pollFn: fn}
}

func TestList(t *testing.T) {
	tests := []struct {
		name      string
		pollers   []poller.Poller
		wantItems int
		wantErr   bool
		validate  func(t *testing.T, items []*workitem.WorkItem)
	}{
		{
			name: "single poller returns items",
			pollers: []poller.Poller{
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return []*workitem.WorkItem{
						workitem.MakeTestItem(workitem.KindPR, "pr.test.repo.1"),
						workitem.MakeTestItem(workitem.KindPR, "pr.test.repo.2"),
					}, nil
				}),
			},
			wantItems: 2,
		},
		{
			name: "multiple pollers concatenate results",
			pollers: []poller.Poller{
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return []*workitem.WorkItem{
						workitem.MakeTestItem(workitem.KindPR, "pr.repo-a.x.1"),
					}, nil
				}),
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return []*workitem.WorkItem{
						workitem.MakeTestItem(workitem.KindPR, "pr.repo-b.x.1"),
					}, nil
				}),
			},
			wantItems: 2,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				if items[0].Name != "pr.repo-a.x.1" {
					t.Errorf("items[0].Name = %q, want %q", items[0].Name, "pr.repo-a.x.1")
				}
				if items[1].Name != "pr.repo-b.x.1" {
					t.Errorf("items[1].Name = %q, want %q", items[1].Name, "pr.repo-b.x.1")
				}
			},
		},
		{
			name: "partial failure returns succeeded results",
			pollers: []poller.Poller{
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return nil, fmt.Errorf("rate limit")
				}),
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return []*workitem.WorkItem{
						workitem.MakeTestItem(workitem.KindPR, "pr.repo-b.x.1"),
					}, nil
				}),
			},
			wantItems: 1,
		},
		{
			name: "all pollers fail returns error",
			pollers: []poller.Poller{
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return nil, fmt.Errorf("error 1")
				}),
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return nil, fmt.Errorf("error 2")
				}),
			},
			wantErr: true,
		},
		{
			name:      "no pollers returns empty",
			pollers:   nil,
			wantItems: 0,
		},
		{
			name: "poller returns empty list",
			pollers: []poller.Poller{
				mockPollerFn(func(_ context.Context) ([]*workitem.WorkItem, error) {
					return nil, nil
				}),
			},
			wantItems: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(logr.Discard(), tt.pollers)
			items, err := e.List(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(items) != tt.wantItems {
					t.Fatalf("items = %d, want %d", len(items), tt.wantItems)
				}
			}

			if tt.validate != nil {
				tt.validate(t, items)
			}
		})
	}
}
