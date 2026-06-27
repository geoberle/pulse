package github

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	gogithub "github.com/google/go-github/v72/github"

	"github.com/geoberle/pulse/internal/workitem"
)

func ptr[T any](v T) *T { return &v }

type mockPRs struct {
	listFn     func(ctx context.Context, owner, repo string, opts *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error)
	commentsFn func(ctx context.Context, owner, repo string, number int, opts *gogithub.PullRequestListCommentsOptions) ([]*gogithub.PullRequestComment, *gogithub.Response, error)
}

func (m *mockPRs) List(ctx context.Context, owner, repo string, opts *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
	return m.listFn(ctx, owner, repo, opts)
}

func (m *mockPRs) ListComments(ctx context.Context, owner, repo string, number int, opts *gogithub.PullRequestListCommentsOptions) ([]*gogithub.PullRequestComment, *gogithub.Response, error) {
	return m.commentsFn(ctx, owner, repo, number, opts)
}

type mockChecks struct {
	listFn func(ctx context.Context, owner, repo, ref string, opts *gogithub.ListCheckRunsOptions) (*gogithub.ListCheckRunsResults, *gogithub.Response, error)
}

func (m *mockChecks) ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *gogithub.ListCheckRunsOptions) (*gogithub.ListCheckRunsResults, *gogithub.Response, error) {
	return m.listFn(ctx, owner, repo, ref, opts)
}

func noComments() func(context.Context, string, string, int, *gogithub.PullRequestListCommentsOptions) ([]*gogithub.PullRequestComment, *gogithub.Response, error) {
	return func(_ context.Context, _, _ string, _ int, _ *gogithub.PullRequestListCommentsOptions) ([]*gogithub.PullRequestComment, *gogithub.Response, error) {
		return nil, &gogithub.Response{}, nil
	}
}

func noChecks() func(context.Context, string, string, string, *gogithub.ListCheckRunsOptions) (*gogithub.ListCheckRunsResults, *gogithub.Response, error) {
	return func(_ context.Context, _, _, _ string, _ *gogithub.ListCheckRunsOptions) (*gogithub.ListCheckRunsResults, *gogithub.Response, error) {
		return &gogithub.ListCheckRunsResults{}, &gogithub.Response{}, nil
	}
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func TestPoll(t *testing.T) {
	tests := []struct {
		name      string
		repos     []string
		user      string
		prs       func(ctx context.Context, owner, repo string, opts *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error)
		comments  func(ctx context.Context, owner, repo string, number int, opts *gogithub.PullRequestListCommentsOptions) ([]*gogithub.PullRequestComment, *gogithub.Response, error)
		checks    func(ctx context.Context, owner, repo, ref string, opts *gogithub.ListCheckRunsOptions) (*gogithub.ListCheckRunsResults, *gogithub.Response, error)
		wantItems int
		wantErr   bool
		validate  func(t *testing.T, items []*workitem.WorkItem)
	}{
		{
			name:  "single PR with no checks or reviews",
			repos: []string{"Azure/ARO-HCP"},
			user:  "testuser",
			prs: func(_ context.Context, _, _ string, _ *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
				return []*gogithub.PullRequest{
					{
						Number: ptr(42),
						Title:  ptr("fix: update reconciler"),
						State:  ptr("open"),
						User:   &gogithub.User{Login: ptr("testuser")},
						Head:   &gogithub.PullRequestBranch{Ref: ptr("fix-reconciler"), SHA: ptr("abc123")},
					},
				}, &gogithub.Response{}, nil
			},
			comments:  noComments(),
			checks:    noChecks(),
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				pr := items[0]
				if pr.Kind != workitem.KindPR {
					t.Errorf("kind = %q, want %q", pr.Kind, workitem.KindPR)
				}
				if pr.ID != "pr:Azure/ARO-HCP:42" {
					t.Errorf("id = %q, want %q", pr.ID, "pr:Azure/ARO-HCP:42")
				}
				if pr.Label != "fix: update reconciler" {
					t.Errorf("label = %q, want %q", pr.Label, "fix: update reconciler")
				}
				if pr.Status != "open" {
					t.Errorf("status = %q, want %q", pr.Status, "open")
				}
				if len(pr.Children) != 0 {
					t.Errorf("children = %d, want 0", len(pr.Children))
				}
				spec := pr.ParsedSpec.(*workitem.PRSpec)
				if spec.Repo != "Azure/ARO-HCP" {
					t.Errorf("spec.Repo = %q, want %q", spec.Repo, "Azure/ARO-HCP")
				}
				if spec.Number != 42 {
					t.Errorf("spec.Number = %d, want 42", spec.Number)
				}
				if spec.Branch != "fix-reconciler" {
					t.Errorf("spec.Branch = %q, want %q", spec.Branch, "fix-reconciler")
				}
			},
		},
		{
			name:  "filters PRs by user",
			repos: []string{"Azure/ARO-HCP"},
			user:  "testuser",
			prs: func(_ context.Context, _, _ string, _ *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
				return []*gogithub.PullRequest{
					{
						Number: ptr(1),
						Title:  ptr("other user's PR"),
						State:  ptr("open"),
						User:   &gogithub.User{Login: ptr("otheruser")},
						Head:   &gogithub.PullRequestBranch{Ref: ptr("other-branch"), SHA: ptr("def456")},
					},
					{
						Number: ptr(2),
						Title:  ptr("my PR"),
						State:  ptr("open"),
						User:   &gogithub.User{Login: ptr("testuser")},
						Head:   &gogithub.PullRequestBranch{Ref: ptr("my-branch"), SHA: ptr("ghi789")},
					},
				}, &gogithub.Response{}, nil
			},
			comments:  noComments(),
			checks:    noChecks(),
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				if items[0].ID != "pr:Azure/ARO-HCP:2" {
					t.Errorf("expected only user's PR, got id %q", items[0].ID)
				}
			},
		},
		{
			name:  "PR with check runs",
			repos: []string{"Azure/ARO-HCP"},
			user:  "testuser",
			prs: func(_ context.Context, _, _ string, _ *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
				return []*gogithub.PullRequest{
					{
						Number: ptr(10),
						Title:  ptr("feat: add tests"),
						State:  ptr("open"),
						User:   &gogithub.User{Login: ptr("testuser")},
						Head:   &gogithub.PullRequestBranch{Ref: ptr("add-tests"), SHA: ptr("sha1")},
					},
				}, &gogithub.Response{}, nil
			},
			comments: noComments(),
			checks: func(_ context.Context, _, _, _ string, _ *gogithub.ListCheckRunsOptions) (*gogithub.ListCheckRunsResults, *gogithub.Response, error) {
				return &gogithub.ListCheckRunsResults{
					CheckRuns: []*gogithub.CheckRun{
						{ID: ptr(int64(100)), Name: ptr("e2e-tests"), Status: ptr("completed"), Conclusion: ptr("success")},
						{ID: ptr(int64(101)), Name: ptr("lint"), Status: ptr("in_progress")},
					},
				}, &gogithub.Response{}, nil
			},
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				pr := items[0]
				if len(pr.Children) != 2 {
					t.Fatalf("children = %d, want 2", len(pr.Children))
				}
				check1 := pr.Children[0]
				if check1.Kind != workitem.KindCheck {
					t.Errorf("child[0].kind = %q, want %q", check1.Kind, workitem.KindCheck)
				}
				if check1.ID != "check:100" {
					t.Errorf("child[0].id = %q, want %q", check1.ID, "check:100")
				}
				if check1.Status != "success" {
					t.Errorf("child[0].status = %q, want %q", check1.Status, "success")
				}

				check2 := pr.Children[1]
				if check2.Status != "in_progress" {
					t.Errorf("child[1].status = %q, want %q", check2.Status, "in_progress")
				}
			},
		},
		{
			name:  "PR with unresolved review comments",
			repos: []string{"Azure/ARO-HCP"},
			user:  "testuser",
			prs: func(_ context.Context, _, _ string, _ *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
				return []*gogithub.PullRequest{
					{
						Number: ptr(20),
						Title:  ptr("feat: new handler"),
						State:  ptr("open"),
						User:   &gogithub.User{Login: ptr("testuser")},
						Head:   &gogithub.PullRequestBranch{Ref: ptr("new-handler"), SHA: ptr("sha2")},
					},
				}, &gogithub.Response{}, nil
			},
			comments: func(_ context.Context, _, _ string, _ int, _ *gogithub.PullRequestListCommentsOptions) ([]*gogithub.PullRequestComment, *gogithub.Response, error) {
				return []*gogithub.PullRequestComment{
					{
						ID:   ptr(int64(500)),
						Body: ptr("nit: rename this variable"),
						Path: ptr("internal/handler.go"),
						User: &gogithub.User{Login: ptr("reviewer1")},
					},
				}, &gogithub.Response{}, nil
			},
			checks:    noChecks(),
			wantItems: 1,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				pr := items[0]
				if len(pr.Children) != 1 {
					t.Fatalf("children = %d, want 1", len(pr.Children))
				}
				review := pr.Children[0]
				if review.Kind != workitem.KindReview {
					t.Errorf("child.kind = %q, want %q", review.Kind, workitem.KindReview)
				}
				if review.ID != "gh-comment:500" {
					t.Errorf("child.id = %q, want %q", review.ID, "gh-comment:500")
				}
				if review.Status != "unresolved" {
					t.Errorf("child.status = %q, want %q", review.Status, "unresolved")
				}
				spec := review.ParsedSpec.(*workitem.ReviewSpec)
				if spec.File != "internal/handler.go" {
					t.Errorf("spec.File = %q, want %q", spec.File, "internal/handler.go")
				}
				wantHash := sha256Hex("nit: rename this variable")
				if spec.BodyHash != wantHash {
					t.Errorf("spec.BodyHash = %q, want %q", spec.BodyHash, wantHash)
				}
			},
		},
		{
			name:  "no open PRs returns empty",
			repos: []string{"Azure/ARO-HCP"},
			user:  "testuser",
			prs: func(_ context.Context, _, _ string, _ *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
				return nil, &gogithub.Response{}, nil
			},
			comments:  noComments(),
			checks:    noChecks(),
			wantItems: 0,
		},
		{
			name:  "multiple repos",
			repos: []string{"Azure/ARO-HCP", "Azure/ARO-Tools"},
			user:  "testuser",
			prs: func(_ context.Context, _, repo string, _ *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
				switch repo {
				case "ARO-HCP":
					return []*gogithub.PullRequest{
						{
							Number: ptr(1),
							Title:  ptr("PR in HCP"),
							State:  ptr("open"),
							User:   &gogithub.User{Login: ptr("testuser")},
							Head:   &gogithub.PullRequestBranch{Ref: ptr("branch-1"), SHA: ptr("sha-a")},
						},
					}, &gogithub.Response{}, nil
				case "ARO-Tools":
					return []*gogithub.PullRequest{
						{
							Number: ptr(2),
							Title:  ptr("PR in Tools"),
							State:  ptr("open"),
							User:   &gogithub.User{Login: ptr("testuser")},
							Head:   &gogithub.PullRequestBranch{Ref: ptr("branch-2"), SHA: ptr("sha-b")},
						},
					}, &gogithub.Response{}, nil
				default:
					return nil, &gogithub.Response{}, nil
				}
			},
			comments:  noComments(),
			checks:    noChecks(),
			wantItems: 2,
			validate: func(t *testing.T, items []*workitem.WorkItem) {
				t.Helper()
				if items[0].ID != "pr:Azure/ARO-HCP:1" {
					t.Errorf("items[0].id = %q, want %q", items[0].ID, "pr:Azure/ARO-HCP:1")
				}
				if items[1].ID != "pr:Azure/ARO-Tools:2" {
					t.Errorf("items[1].id = %q, want %q", items[1].ID, "pr:Azure/ARO-Tools:2")
				}
			},
		},
		{
			name:  "API error propagates",
			repos: []string{"Azure/ARO-HCP"},
			user:  "testuser",
			prs: func(_ context.Context, _, _ string, _ *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error) {
				return nil, nil, fmt.Errorf("rate limit exceeded")
			},
			comments:  noComments(),
			checks:    noChecks(),
			wantItems: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPoller(
				&mockPRs{listFn: tt.prs, commentsFn: tt.comments},
				&mockChecks{listFn: tt.checks},
				tt.repos,
				tt.user,
			)

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

func TestFindUnresolvedComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []*gogithub.PullRequestComment
		user     string
		wantIDs  []int64
	}{
		{
			name:     "no comments",
			comments: nil,
			user:     "me",
			wantIDs:  nil,
		},
		{
			name: "reviewer comment with no reply is unresolved",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("fix this"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}},
			},
			user:    "me",
			wantIDs: []int64{1},
		},
		{
			name: "user's own comment is not unresolved",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("note to self"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}},
			},
			user:    "me",
			wantIDs: nil,
		},
		{
			name: "reviewer comment with user reply is resolved",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("fix this"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}},
				{ID: ptr(int64(2)), Body: ptr("done"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}, InReplyTo: ptr(int64(1))},
			},
			user:    "me",
			wantIDs: nil,
		},
		{
			name: "multiple threads mixed resolved and unresolved",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("issue A"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}},
				{ID: ptr(int64(2)), Body: ptr("fixed A"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}, InReplyTo: ptr(int64(1))},
				{ID: ptr(int64(3)), Body: ptr("issue B"), Path: ptr("b.go"), User: &gogithub.User{Login: ptr("reviewer")}},
			},
			user:    "me",
			wantIDs: []int64{3},
		},
		{
			name: "nested reply from user resolves thread",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("root comment"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}},
				{ID: ptr(int64(2)), Body: ptr("followup"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}, InReplyTo: ptr(int64(1))},
				{ID: ptr(int64(3)), Body: ptr("addressed"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}, InReplyTo: ptr(int64(2))},
			},
			user:    "me",
			wantIDs: nil,
		},
		{
			name: "copilot reviewer comment is unresolved same as human",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("suggestion"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("copilot-pull-request-reviewer[bot]")}},
			},
			user:    "me",
			wantIDs: []int64{1},
		},
		{
			name: "user-started thread with reviewer reply needs attention",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("what about X?"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}},
				{ID: ptr(int64(2)), Body: ptr("no, use Y"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}, InReplyTo: ptr(int64(1))},
			},
			user:    "me",
			wantIDs: []int64{1},
		},
		{
			name: "user-started thread resolved when user replies after reviewer",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("what about X?"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}},
				{ID: ptr(int64(2)), Body: ptr("no, use Y"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}, InReplyTo: ptr(int64(1))},
				{ID: ptr(int64(3)), Body: ptr("ok done"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}, InReplyTo: ptr(int64(1))},
			},
			user:    "me",
			wantIDs: nil,
		},
		{
			name: "resolved thread becomes unresolved when reviewer replies again",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("fix this"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}},
				{ID: ptr(int64(2)), Body: ptr("done"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}, InReplyTo: ptr(int64(1))},
				{ID: ptr(int64(3)), Body: ptr("still wrong"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("reviewer")}, InReplyTo: ptr(int64(1))},
			},
			user:    "me",
			wantIDs: []int64{1},
		},
		{
			name: "orphaned reply from reviewer treated as own thread root",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(5)), Body: ptr("this is wrong"), Path: ptr("b.go"), User: &gogithub.User{Login: ptr("reviewer")}, InReplyTo: ptr(int64(999))},
			},
			user:    "me",
			wantIDs: []int64{5},
		},
		{
			name: "user-only thread with no reviewer interaction is not unresolved",
			comments: []*gogithub.PullRequestComment{
				{ID: ptr(int64(1)), Body: ptr("note"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}},
				{ID: ptr(int64(2)), Body: ptr("followup"), Path: ptr("a.go"), User: &gogithub.User{Login: ptr("me")}, InReplyTo: ptr(int64(1))},
			},
			user:    "me",
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findUnresolvedComments(tt.comments, tt.user)

			gotIDs := make(map[int64]bool)
			for _, c := range got {
				gotIDs[c.GetID()] = true
			}

			wantIDSet := make(map[int64]bool)
			for _, id := range tt.wantIDs {
				wantIDSet[id] = true
			}

			if len(gotIDs) != len(wantIDSet) {
				t.Fatalf("got %d unresolved comments, want %d", len(gotIDs), len(wantIDSet))
			}
			for id := range wantIDSet {
				if !gotIDs[id] {
					t.Errorf("expected comment %d to be unresolved", id)
				}
			}
		})
	}
}

func TestBodyHash(t *testing.T) {
	hash := bodyHash("test body")
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	hash2 := bodyHash("test body")
	if hash != hash2 {
		t.Error("same input produced different hashes")
	}

	hash3 := bodyHash("different body")
	if hash == hash3 {
		t.Error("different inputs produced same hash")
	}
}
