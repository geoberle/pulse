package jira

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
				if item.Name != "jira.aro-100" {
					t.Errorf("Name = %q, want %q", item.Name, "jira.aro-100")
				}
				if item.DisplayName() != "fix auth" {
					t.Errorf("DisplayName = %q, want %q", item.DisplayName(), "fix auth")
				}
				if item.Status.Phase != "In Progress" {
					t.Errorf("Status.Phase = %q, want %q", item.Status.Phase, "In Progress")
				}
				if item.Kind != string(workitem.KindJira) {
					t.Errorf("Kind = %q, want %q", item.Kind, string(workitem.KindJira))
				}
				spec, ok := item.ParsedSpec.(*workitem.JiraSpec)
				if !ok {
					t.Fatalf("ParsedSpec type = %T, want *workitem.JiraSpec", item.ParsedSpec)
				}
				if spec.Key != "ARO-100" {
					t.Errorf("spec.Key = %q, want %q", spec.Key, "ARO-100")
				}
				cond := meta.FindStatusCondition(item.Status.Conditions, string(workitem.ConditionStale))
				if cond == nil {
					t.Fatal("expected Stale condition to be set")
				}
				if cond.Status != metav1.ConditionFalse {
					t.Errorf("Stale condition status = %q, want %q", cond.Status, metav1.ConditionFalse)
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
				cond := meta.FindStatusCondition(items[0].Status.Conditions, string(workitem.ConditionStale))
				if cond == nil {
					t.Fatal("expected Stale condition to be set")
				}
				if cond.Status != metav1.ConditionTrue {
					t.Errorf("Stale condition status = %q, want %q", cond.Status, metav1.ConditionTrue)
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
				if items[0].Name != "jira.aro-1" {
					t.Errorf("items[0].Name = %q, want %q", items[0].Name, "jira.aro-1")
				}
				if items[1].Name != "jira.aro-2" {
					t.Errorf("items[1].Name = %q, want %q", items[1].Name, "jira.aro-2")
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
			name: "nil updated field has no stale condition",
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
				cond := meta.FindStatusCondition(items[0].Status.Conditions, string(workitem.ConditionStale))
				if cond != nil {
					t.Errorf("expected no Stale condition when Updated is nil, got %v", cond)
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
			wantErr: true,
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
				if items[0].DisplayName() != "has summary" {
					t.Errorf("DisplayName = %q, want %q", items[0].DisplayName(), "has summary")
				}
				if items[0].Status.Phase != "" {
					t.Errorf("Status.Phase = %q, want empty", items[0].Status.Phase)
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
			name: "empty issue key returns validation error",
			searcher: &mockSearcher{searchFn: func(_ context.Context, _ string, _, _ []string, _ int, _ string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error) {
				return &models.IssueSearchJQLSchemeV2{
					Issues: []*models.IssueSchemeV2{
						{
							Key: "",
							Fields: &models.IssueFieldsSchemeV2{
								Summary: "bad issue",
								Status:  &models.StatusScheme{Name: "Open"},
							},
						},
					},
				}, nil, nil
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
