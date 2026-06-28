package jira

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"

	"github.com/geoberle/pulse/internal/workitem"
)

type mockSearcher struct {
	searchFn func(ctx context.Context, jql string, fields, expands []string, maxResults int, nextPageToken string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error)
}

func (m *mockSearcher) SearchJQL(ctx context.Context, jql string, fields, expands []string, maxResults int, nextPageToken string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
	return m.searchFn(ctx, jql, fields, expands, maxResults, nextPageToken)
}

func dateTime(t time.Time) *models.DateTimeScheme {
	dt := models.DateTimeScheme(t)
	return &dt
}

func TestPoll(t *testing.T) {
	fixedNow := time.Date(2025, 6, 28, 12, 0, 0, 0, time.UTC)
	staleThreshold := 120 * time.Hour // 5 days

	tests := []struct {
		name      string
		project   string
		searcher  IssueSearcher
		wantItems int
		wantErr   bool
		validate  func(t *testing.T, items []*workitem.WorkItem)
	}{
		{
			name: "single issue active",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{
							Key: "ARO-100",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "fix auth",
								Status:  &models.StatusScheme{Name: "In Progress"},
								Updated: dateTime(fixedNow.Add(-24 * time.Hour)),
							},
						},
					},
				}, nil, nil
			}},
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				item := items[0]
				if item.ID != "jira:ARO-100" {
					t.Errorf("ID = %q, want %q", item.ID, "jira:ARO-100")
				}
				if item.Label != "fix auth" {
					t.Errorf("Label = %q, want %q", item.Label, "fix auth")
				}
				if item.Status != "In Progress" {
					t.Errorf("Status = %q, want %q", item.Status, "In Progress")
				}
				if item.Kind != workitem.KindJira {
					t.Errorf("Kind = %q, want %q", item.Kind, workitem.KindJira)
				}
				spec, ok := item.ParsedSpec.(*workitem.JiraSpec)
				if !ok {
					t.Fatalf("ParsedSpec type = %T, want *workitem.JiraSpec", item.ParsedSpec)
				}
				if spec.Key != "ARO-100" {
					t.Errorf("spec.Key = %q, want %q", spec.Key, "ARO-100")
				}
				if spec.Staleness != workitem.StalenessActive {
					t.Errorf("spec.Staleness = %q, want %q", spec.Staleness, workitem.StalenessActive)
				}
			},
		},
		{
			name: "stale issue detected",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{
							Key: "ARO-200",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "old task",
								Status:  &models.StatusScheme{Name: "To Do"},
								Updated: dateTime(fixedNow.Add(-200 * time.Hour)),
							},
						},
					},
				}, nil, nil
			}},
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				spec := items[0].ParsedSpec.(*workitem.JiraSpec)
				if spec.Staleness != workitem.StalenessStale {
					t.Errorf("spec.Staleness = %q, want %q", spec.Staleness, workitem.StalenessStale)
				}
			},
		},
		{
			name: "multiple issues",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{
							Key: "ARO-1",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "first",
								Status:  &models.StatusScheme{Name: "Open"},
								Updated: dateTime(fixedNow),
							},
						},
						{
							Key: "ARO-2",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "second",
								Status:  &models.StatusScheme{Name: "Done"},
								Updated: dateTime(fixedNow),
							},
						},
					},
				}, nil, nil
			}},
			wantItems: 2,
		},
		{
			name: "pagination fetches all pages",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, nextPageToken string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				if len(nextPageToken) == 0 {
					return &models.IssueSearchJQLSchemeV2{
						Issues: []*models.IssueSchemeV2{
							{
								Key: "ARO-1",
								Fields: &models.IssueFieldsSchemeV2{
									Summary: "page 1",
									Status:  &models.StatusScheme{Name: "Open"},
									Updated: dateTime(fixedNow),
								},
							},
						},
						NextPageToken: "page2token",
					}, nil, nil
				}
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{
							Key: "ARO-2",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "page 2",
								Status:  &models.StatusScheme{Name: "Open"},
								Updated: dateTime(fixedNow),
							},
						},
					},
				}, nil, nil
			}},
			wantItems: 2,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				if items[0].ID != "jira:ARO-1" {
					t.Errorf("items[0].ID = %q, want %q", items[0].ID, "jira:ARO-1")
				}
				if items[1].ID != "jira:ARO-2" {
					t.Errorf("items[1].ID = %q, want %q", items[1].ID, "jira:ARO-2")
				}
			},
		},
		{
			name: "empty result",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{}, nil, nil
			}},
			wantItems: 0,
		},
		{
			name: "api error",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return nil, &models.ResponseScheme{Code: 401}, fmt.Errorf("unauthorized")
			}},
			wantErr: true,
		},
		{
			name: "nil updated field leaves staleness unknown",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{
							Key: "ARO-300",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "no update time",
								Status:  &models.StatusScheme{Name: "Open"},
							},
						},
					},
				}, nil, nil
			}},
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				spec := items[0].ParsedSpec.(*workitem.JiraSpec)
				if spec.Staleness != workitem.StalenessUnknown {
					t.Errorf("spec.Staleness = %q, want %q", spec.Staleness, workitem.StalenessUnknown)
				}
			},
		},
		{
			name: "nil fields produces empty summary and status",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{Key: "ARO-400"},
					},
				}, nil, nil
			}},
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				if items[0].Label != "" {
					t.Errorf("Label = %q, want empty", items[0].Label)
				}
				if items[0].Status != "" {
					t.Errorf("Status = %q, want empty", items[0].Status)
				}
				spec := items[0].ParsedSpec.(*workitem.JiraSpec)
				if spec.Staleness != workitem.StalenessUnknown {
					t.Errorf("Staleness = %q, want %q", spec.Staleness, workitem.StalenessUnknown)
				}
			},
		},
		{
			name: "nil status within non-nil fields",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{
							Key: "ARO-500",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "has summary",
								Updated: dateTime(fixedNow),
							},
						},
					},
				}, nil, nil
			}},
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				if items[0].Label != "has summary" {
					t.Errorf("Label = %q, want %q", items[0].Label, "has summary")
				}
				if items[0].Status != "" {
					t.Errorf("Status = %q, want empty", items[0].Status)
				}
			},
		},
		{
			name: "api error with nil response",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return nil, nil, fmt.Errorf("connection refused")
			}},
			wantErr: true,
		},
		{
			name:    "uses correct JQL with project",
			project: "MYPROJ",
			searcher: &mockSearcher{searchFn: func(_ context.Context, jql string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				want := `project = MYPROJ AND (assignee = currentUser() OR creator = currentUser()) AND type in (Story, Bug) AND status != Closed`
				if jql != want {
					return nil, nil, fmt.Errorf("JQL = %q, want %q", jql, want)
				}
				return &models.IssueSearchJQLSchemeV2{}, nil, nil
			}},
			wantItems: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			project := tt.project
			if len(project) == 0 {
				project = "ARO"
			}

			p := NewPoller(tt.searcher, project, staleThreshold)
			p.now = func() time.Time { return fixedNow }

			items, err := p.Poll(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != tt.wantItems {
				t.Fatalf("items = %d, want %d", len(items), tt.wantItems)
			}
			if tt.validate != nil {
				tt.validate(t, items)
			}
		})
	}
}
