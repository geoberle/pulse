package storage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/geoberle/pulse/internal/api"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestWorktreesCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	c := s.Worktrees()

	now := time.Now().UTC()
	wt := &api.Worktree{
		Path: "/home/user/dev/repo.branch", Repo: "Azure/ARO-HCP",
		Branch: "feature", CommitState: api.WorktreeCommitStateHasCommits, LastSeen: now,
	}

	// create
	created, err := c.Create(ctx, wt)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceVersion != 1 {
		t.Errorf("expected RV 1, got %d", created.ResourceVersion)
	}
	if created.CreationTimestamp.IsZero() {
		t.Error("expected non-zero CreationTimestamp")
	}
	if created.Path != wt.Path {
		t.Errorf("expected path %s, got %s", wt.Path, created.Path)
	}
	if created.CommitState != api.WorktreeCommitStateHasCommits {
		t.Errorf("expected CommitState=%q, got %q", api.WorktreeCommitStateHasCommits, created.CommitState)
	}

	// get
	got, err := c.Get(ctx, wt.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Repo != "Azure/ARO-HCP" {
		t.Errorf("expected repo Azure/ARO-HCP, got %s", got.Repo)
	}

	// update
	got.Branch = "feature-v2"
	updated, err := c.Update(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResourceVersion != 2 {
		t.Errorf("expected RV 2, got %d", updated.ResourceVersion)
	}
	if updated.Branch != "feature-v2" {
		t.Errorf("expected branch feature-v2, got %s", updated.Branch)
	}

	// optimistic concurrency: stale update
	got.Branch = "stale"
	_, err = c.Update(ctx, got)
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// list
	items, err := c.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// delete
	if err := c.Delete(ctx, wt.Path); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(ctx, wt.Path)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// delete nonexistent
	err = c.Delete(ctx, "/nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPullRequestsCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	c := s.PullRequests()

	now := time.Now().UTC()
	pr := &api.PullRequest{
		Repo: "Azure/ARO-HCP", Number: 42, Branch: "feature",
		URL: "https://github.com/Azure/ARO-HCP/pull/42", Status: api.PullRequestStatusOpen,
		CIStatus: api.CIStatusPassing, ReviewStatus: api.ReviewStatusPending,
		UnresolvedComments: 2, Author: "geoberle", LastSeen: now,
	}

	// create
	created, err := c.Create(ctx, pr)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceVersion != 1 {
		t.Errorf("expected RV 1, got %d", created.ResourceVersion)
	}
	if created.Key() != "Azure/ARO-HCP#42" {
		t.Errorf("expected key Azure/ARO-HCP#42, got %s", created.Key())
	}

	// get
	got, err := c.Get(ctx, "Azure/ARO-HCP#42")
	if err != nil {
		t.Fatal(err)
	}
	if got.UnresolvedComments != 2 {
		t.Errorf("expected 2 unresolved comments, got %d", got.UnresolvedComments)
	}

	// update
	got.Status = api.PullRequestStatusMerged
	got.CIStatus = api.CIStatusPassing
	updated, err := c.Update(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResourceVersion != 2 {
		t.Errorf("expected RV 2, got %d", updated.ResourceVersion)
	}

	// stale update
	got.Status = api.PullRequestStatusClosed
	_, err = c.Update(ctx, got)
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// list
	items, err := c.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// delete
	if err := c.Delete(ctx, "Azure/ARO-HCP#42"); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(ctx, "Azure/ARO-HCP#42")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestJiraTicketsCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	c := s.JiraTickets()

	now := time.Now().UTC()
	jt := &api.JiraTicket{
		TicketKey: "ARO-12345", Summary: "Fix fleet caching",
		Description: "Details here", Status: api.JiraStatus("In Progress"),
		IssueType: api.JiraIssueType("Story"), EpicKey: "ARO-10000",
		LastActivity: now, LastSeen: now,
	}

	// create
	created, err := c.Create(ctx, jt)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceVersion != 1 {
		t.Errorf("expected RV 1, got %d", created.ResourceVersion)
	}
	if created.Key() != "ARO-12345" {
		t.Errorf("expected key ARO-12345, got %s", created.Key())
	}

	// get
	got, err := c.Get(ctx, "ARO-12345")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "Fix fleet caching" {
		t.Errorf("expected summary 'Fix fleet caching', got %s", got.Summary)
	}

	// update
	got.Status = api.JiraStatus("Done")
	updated, err := c.Update(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResourceVersion != 2 {
		t.Errorf("expected RV 2, got %d", updated.ResourceVersion)
	}

	// stale update
	got.Status = api.JiraStatus("Reopened")
	_, err = c.Update(ctx, got)
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// delete
	if err := c.Delete(ctx, "ARO-12345"); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(ctx, "ARO-12345")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManualLinksCRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	c := s.ManualLinks()

	ml := &api.ManualLink{
		SourceType: api.ManualLinkSourceTypeWorktree, SourceID: "/home/user/dev/repo.branch",
		JiraKey: "ARO-12345",
	}

	// create
	created, err := c.Create(ctx, ml)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceVersion != 1 {
		t.Errorf("expected RV 1, got %d", created.ResourceVersion)
	}
	if created.Key() != "worktree//home/user/dev/repo.branch" {
		t.Errorf("expected key worktree//home/user/dev/repo.branch, got %s", created.Key())
	}

	// get
	got, err := c.Get(ctx, "worktree//home/user/dev/repo.branch")
	if err != nil {
		t.Fatal(err)
	}
	if got.JiraKey != "ARO-12345" {
		t.Errorf("expected jira key ARO-12345, got %s", got.JiraKey)
	}

	// update
	got.JiraKey = "ARO-99999"
	updated, err := c.Update(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResourceVersion != 2 {
		t.Errorf("expected RV 2, got %d", updated.ResourceVersion)
	}
	if updated.JiraKey != "ARO-99999" {
		t.Errorf("expected jira key ARO-99999, got %s", updated.JiraKey)
	}

	// stale update
	got.JiraKey = "ARO-11111"
	_, err = c.Update(ctx, got)
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// delete
	if err := c.Delete(ctx, "worktree//home/user/dev/repo.branch"); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(ctx, "worktree//home/user/dev/repo.branch")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	now := time.Now().UTC()

	tests := []struct {
		name    string
		fn      func() error
		wantErr string
	}{
		{
			name: "worktree missing path",
			fn: func() error {
				_, err := s.Worktrees().Create(ctx, &api.Worktree{
					Repo: "org/repo", Branch: "main", LastSeen: now,
				})
				return err
			},
			wantErr: "path is required",
		},
		{
			name: "worktree invalid commit state",
			fn: func() error {
				_, err := s.Worktrees().Create(ctx, &api.Worktree{
					Path: "/tmp/wt", Repo: "org/repo", Branch: "main",
					CommitState: "bogus", LastSeen: now,
				})
				return err
			},
			wantErr: "invalid commit state",
		},
		{
			name: "pull request invalid status",
			fn: func() error {
				_, err := s.PullRequests().Create(ctx, &api.PullRequest{
					Repo: "org/repo", Number: 1, Branch: "main",
					URL: "https://example.com", Status: "bogus",
					CIStatus: api.CIStatusPending, ReviewStatus: api.ReviewStatusPending,
					Author: "user", LastSeen: now,
				})
				return err
			},
			wantErr: "invalid status",
		},
		{
			name: "pull request invalid ci status",
			fn: func() error {
				_, err := s.PullRequests().Create(ctx, &api.PullRequest{
					Repo: "org/repo", Number: 2, Branch: "main",
					URL: "https://example.com", Status: api.PullRequestStatusOpen,
					CIStatus: "bogus", ReviewStatus: api.ReviewStatusPending,
					Author: "user", LastSeen: now,
				})
				return err
			},
			wantErr: "invalid ci status",
		},
		{
			name: "jira ticket empty status",
			fn: func() error {
				_, err := s.JiraTickets().Create(ctx, &api.JiraTicket{
					TicketKey: "ARO-1", Summary: "test",
					Status: "", IssueType: "Bug",
					LastActivity: now, LastSeen: now,
				})
				return err
			},
			wantErr: "status is required",
		},
		{
			name: "jira ticket empty issue type",
			fn: func() error {
				_, err := s.JiraTickets().Create(ctx, &api.JiraTicket{
					TicketKey: "ARO-2", Summary: "test",
					Status: "To Do", IssueType: "",
					LastActivity: now, LastSeen: now,
				})
				return err
			},
			wantErr: "issue type is required",
		},
		{
			name: "manual link invalid source type",
			fn: func() error {
				_, err := s.ManualLinks().Create(ctx, &api.ManualLink{
					SourceType: "bogus", SourceID: "id", JiraKey: "ARO-1",
				})
				return err
			},
			wantErr: "invalid source type",
		},
		{
			name: "manual link empty jira key",
			fn: func() error {
				_, err := s.ManualLinks().Create(ctx, &api.ManualLink{
					SourceType: api.ManualLinkSourceTypeWorktree,
					SourceID:   "/tmp/wt", JiraKey: "",
				})
				return err
			},
			wantErr: "jira key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestUpdateNonexistent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)

	_, err := s.Worktrees().Update(ctx, &api.Worktree{
		ObjectMeta: api.ObjectMeta{ResourceVersion: 1},
		Path:       "/nonexistent",
		Repo:       "org/repo",
		Branch:     "main",
		LastSeen:   time.Now(),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for nonexistent update, got %v", err)
	}
}
