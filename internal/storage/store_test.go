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

	wt := &api.Worktree{
		ObjectMeta:  api.ObjectMeta{Name: "/home/user/dev/repo.branch"},
		Repo:        "Azure/ARO-HCP",
		Branch:      "feature",
		CommitState: api.WorktreeCommitStateHasCommits,
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
	if created.Name != wt.Name {
		t.Errorf("expected name %s, got %s", wt.Name, created.Name)
	}
	if created.CommitState != api.WorktreeCommitStateHasCommits {
		t.Errorf("expected CommitState=%q, got %q", api.WorktreeCommitStateHasCommits, created.CommitState)
	}

	// get
	got, err := c.Get(ctx, wt.Name)
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
	if err := c.Delete(ctx, wt.Name); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(ctx, wt.Name)
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

	pr := &api.PullRequest{
		Repo:               "Azure/ARO-HCP",
		Number:             42,
		Branch:             "feature",
		URL:                "https://github.com/Azure/ARO-HCP/pull/42",
		Status:             api.PullRequestStatusOpen,
		CIStatus:           api.CIStatusPassing,
		ReviewStatus:       api.ReviewStatusPending,
		UnresolvedComments: 2,
		Author:             "geoberle",
	}

	// create
	created, err := c.Create(ctx, pr)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceVersion != 1 {
		t.Errorf("expected RV 1, got %d", created.ResourceVersion)
	}
	if created.Name != "Azure/ARO-HCP#42" {
		t.Errorf("expected name Azure/ARO-HCP#42, got %s", created.Name)
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
		ObjectMeta:   api.ObjectMeta{Name: "ARO-12345"},
		Summary:      "Fix fleet caching",
		Description:  "Details here",
		Status:       api.JiraStatus("In Progress"),
		IssueType:    api.JiraIssueType("Story"),
		EpicKey:      "ARO-10000",
		LastActivity: now,
	}

	// create
	created, err := c.Create(ctx, jt)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceVersion != 1 {
		t.Errorf("expected RV 1, got %d", created.ResourceVersion)
	}
	if created.Name != "ARO-12345" {
		t.Errorf("expected name ARO-12345, got %s", created.Name)
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

	mlName := api.ManualLinkName(api.ManualLinkSourceTypeWorktree, "/home/user/dev/repo.branch")
	ml := &api.ManualLink{
		SourceType: api.ManualLinkSourceTypeWorktree,
		SourceID:   "/home/user/dev/repo.branch",
		JiraKey:    "ARO-12345",
	}

	// create
	created, err := c.Create(ctx, ml)
	if err != nil {
		t.Fatal(err)
	}
	if created.ResourceVersion != 1 {
		t.Errorf("expected RV 1, got %d", created.ResourceVersion)
	}
	if created.Name != "worktree//home/user/dev/repo.branch" {
		t.Errorf("expected name worktree//home/user/dev/repo.branch, got %s", created.Name)
	}

	// get
	got, err := c.Get(ctx, mlName)
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
	if err := c.Delete(ctx, mlName); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(ctx, mlName)
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
			name: "worktree missing name",
			fn: func() error {
				_, err := s.Worktrees().Create(ctx, &api.Worktree{
					Repo: "org/repo", Branch: "main",
				})
				return err
			},
			wantErr: "name is required",
		},
		{
			name: "worktree invalid commit state",
			fn: func() error {
				_, err := s.Worktrees().Create(ctx, &api.Worktree{
					ObjectMeta:  api.ObjectMeta{Name: "/tmp/wt"},
					Repo:        "org/repo",
					Branch:      "main",
					CommitState: "bogus",
				})
				return err
			},
			wantErr: "invalid commit state",
		},
		{
			name: "pull request missing repo",
			fn: func() error {
				_, err := s.PullRequests().Create(ctx, &api.PullRequest{
					Number: 1, Branch: "main",
					URL: "https://example.com", Status: api.PullRequestStatusOpen,
					CIStatus: api.CIStatusPending, ReviewStatus: api.ReviewStatusPending,
					Author: "user",
				})
				return err
			},
			wantErr: "repo is required",
		},
		{
			name: "pull request invalid status",
			fn: func() error {
				_, err := s.PullRequests().Create(ctx, &api.PullRequest{
					Repo: "org/repo", Number: 1, Branch: "main",
					URL: "https://example.com", Status: "bogus",
					CIStatus: api.CIStatusPending, ReviewStatus: api.ReviewStatusPending,
					Author: "user",
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
					Author: "user",
				})
				return err
			},
			wantErr: "invalid ci status",
		},
		{
			name: "jira ticket missing name",
			fn: func() error {
				_, err := s.JiraTickets().Create(ctx, &api.JiraTicket{
					Summary: "test", Status: "To Do", IssueType: "Bug",
					LastActivity: now,
				})
				return err
			},
			wantErr: "name is required",
		},
		{
			name: "jira ticket empty status",
			fn: func() error {
				_, err := s.JiraTickets().Create(ctx, &api.JiraTicket{
					ObjectMeta: api.ObjectMeta{Name: "ARO-1"},
					Summary:    "test",
					Status:     "", IssueType: "Bug",
					LastActivity: now,
				})
				return err
			},
			wantErr: "status is required",
		},
		{
			name: "jira ticket empty issue type",
			fn: func() error {
				_, err := s.JiraTickets().Create(ctx, &api.JiraTicket{
					ObjectMeta: api.ObjectMeta{Name: "ARO-2"},
					Summary:    "test",
					Status:     "To Do", IssueType: "",
					LastActivity: now,
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
		{
			name: "duplicate finalizers",
			fn: func() error {
				_, err := s.Worktrees().Create(ctx, &api.Worktree{
					ObjectMeta: api.ObjectMeta{
						Name:       "/tmp/dup-fin",
						Finalizers: []string{"a", "a"},
					},
					Repo: "org/repo", Branch: "main",
				})
				return err
			},
			wantErr: "duplicate finalizer",
		},
		{
			name: "empty finalizer",
			fn: func() error {
				_, err := s.Worktrees().Create(ctx, &api.Worktree{
					ObjectMeta: api.ObjectMeta{
						Name:       "/tmp/empty-fin",
						Finalizers: []string{""},
					},
					Repo: "org/repo", Branch: "main",
				})
				return err
			},
			wantErr: "empty finalizer",
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
		ObjectMeta: api.ObjectMeta{Name: "/nonexistent", ResourceVersion: 1},
		Repo:       "org/repo",
		Branch:     "main",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for nonexistent update, got %v", err)
	}
}

func TestDeleteSoftDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	c := s.Worktrees()

	wt := &api.Worktree{
		ObjectMeta: api.ObjectMeta{
			Name:       "/tmp/finalized",
			Finalizers: []string{"controller-a"},
		},
		Repo:   "org/repo",
		Branch: "main",
	}
	created, err := c.Create(ctx, wt)
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Finalizers) != 1 {
		t.Fatalf("expected 1 finalizer, got %d", len(created.Finalizers))
	}

	// delete with finalizers → soft delete
	if err := c.Delete(ctx, wt.Name); err != nil {
		t.Fatal(err)
	}

	// object still exists with DeletionTimestamp set
	got, err := c.Get(ctx, wt.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.DeletionTimestamp == nil {
		t.Fatal("expected DeletionTimestamp to be set")
	}
	if got.ResourceVersion != 2 {
		t.Errorf("expected RV 2 after soft delete, got %d", got.ResourceVersion)
	}

	// double delete is no-op
	if err := c.Delete(ctx, wt.Name); err != nil {
		t.Fatalf("expected no-op on double delete, got %v", err)
	}

	// remove finalizer via update → garbage collection
	got.Finalizers = nil
	_, err = c.Update(ctx, got)
	if err != nil {
		t.Fatal(err)
	}

	// object is now gone
	_, err = c.Get(ctx, wt.Name)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after garbage collection, got %v", err)
	}
}

func TestDeleteSoftDeleteRowsAffectedCheck(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)

	// Verify RowsAffected check works by creating an object with finalizers,
	// soft-deleting it (succeeds), then verifying double-delete is a no-op
	// (returns nil because DeletionTimestamp already set).
	c := s.Worktrees()
	wt := &api.Worktree{
		ObjectMeta: api.ObjectMeta{
			Name:       "/tmp/rows-check",
			Finalizers: []string{"controller-a"},
		},
		Repo:   "org/repo",
		Branch: "main",
	}
	if _, err := c.Create(ctx, wt); err != nil {
		t.Fatal(err)
	}

	if err := c.Delete(ctx, wt.Name); err != nil {
		t.Fatal(err)
	}

	// verify soft-delete bumped RV and set DeletionTimestamp
	got, err := c.Get(ctx, wt.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.ResourceVersion != 2 {
		t.Errorf("expected RV 2, got %d", got.ResourceVersion)
	}
	if got.DeletionTimestamp == nil {
		t.Fatal("expected DeletionTimestamp set")
	}

	// second delete is no-op (already has DeletionTimestamp)
	if err := c.Delete(ctx, wt.Name); err != nil {
		t.Errorf("expected no-op, got %v", err)
	}
}

func TestUpdateRemoveFinalizerWithFieldChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	c := s.Worktrees()

	wt := &api.Worktree{
		ObjectMeta: api.ObjectMeta{
			Name:       "/tmp/gc-with-change",
			Finalizers: []string{"controller-a"},
		},
		Repo:   "org/repo",
		Branch: "main",
	}
	if _, err := c.Create(ctx, wt); err != nil {
		t.Fatal(err)
	}

	// soft delete
	if err := c.Delete(ctx, wt.Name); err != nil {
		t.Fatal(err)
	}

	// get the soft-deleted object, modify a field AND remove finalizer
	got, err := c.Get(ctx, wt.Name)
	if err != nil {
		t.Fatal(err)
	}
	got.Branch = "cleanup-branch"
	got.Finalizers = nil

	updated, err := c.Update(ctx, got)
	if err != nil {
		t.Fatal(err)
	}

	// should have been garbage collected
	if updated.DeletionTimestamp == nil {
		t.Error("expected DeletionTimestamp on returned object")
	}
	if updated.Branch != "cleanup-branch" {
		t.Errorf("expected branch cleanup-branch, got %s", updated.Branch)
	}

	// object gone from DB
	_, err = c.Get(ctx, wt.Name)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after GC, got %v", err)
	}
}

func TestDeleteNoFinalizers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	c := s.Worktrees()

	wt := &api.Worktree{
		ObjectMeta: api.ObjectMeta{Name: "/tmp/no-finalizers"},
		Repo:       "org/repo",
		Branch:     "main",
	}
	if _, err := c.Create(ctx, wt); err != nil {
		t.Fatal(err)
	}

	// delete without finalizers → hard delete
	if err := c.Delete(ctx, wt.Name); err != nil {
		t.Fatal(err)
	}

	_, err := c.Get(ctx, wt.Name)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after hard delete, got %v", err)
	}
}
