package workitem

import (
	"fmt"
	"strings"
)

// StalenessState represents whether a Jira issue has gone stale based on
// the configured threshold. Zero value ("") means not yet evaluated and
// is omitted from JSON via omitempty — presence implies evaluated.
type StalenessState string

const (
	StalenessUnknown StalenessState = ""
	StalenessActive  StalenessState = "Active"
	StalenessStale   StalenessState = "Stale"
)

// Validate checks that the StalenessState is a known value.
func (s StalenessState) Validate() error {
	switch s {
	case StalenessUnknown, StalenessActive, StalenessStale:
		return nil
	default:
		return fmt.Errorf("unknown staleness state %q", s)
	}
}

// BranchState represents the state of a PR branch relative to its target.
// Zero value ("") means not yet checked and is omitted from JSON via
// omitempty — presence implies checked.
type BranchState string

const (
	BranchStateUnknown     BranchState = ""
	BranchStateUpToDate    BranchState = "UpToDate"
	BranchStateNeedsRebase BranchState = "NeedsRebase"
)

// Validate checks that the BranchState is a known value.
func (b BranchState) Validate() error {
	switch b {
	case BranchStateUnknown, BranchStateUpToDate, BranchStateNeedsRebase:
		return nil
	default:
		return fmt.Errorf("unknown branch state %q", b)
	}
}

var (
	_ Spec = (*JiraSpec)(nil)
	_ Spec = (*PRSpec)(nil)
	_ Spec = (*CheckSpec)(nil)
	_ Spec = (*ReviewSpec)(nil)
	_ Spec = (*LocalSpec)(nil)
)

// JiraSpec holds type-specific fields for a Jira work item (Story or Bug).
type JiraSpec struct {
	// Key is the Jira issue key. Required. Format: "PROJECT-NUMBER".
	// Example: "ARO-12345"
	Key string `json:"key"`

	// Staleness indicates whether the issue has been inactive beyond
	// stale_threshold. Must be a valid StalenessState value.
	// Empty (StalenessUnknown) means not yet evaluated.
	Staleness StalenessState `json:"staleness,omitempty"`
}

// Validate checks required fields, length constraints, and enum values.
func (s *JiraSpec) Validate() error {
	if len(s.Key) == 0 {
		return fmt.Errorf("key is required")
	}
	if len(s.Key) > 50 {
		return fmt.Errorf("key: max 50 chars, got %d", len(s.Key))
	}
	if !strings.Contains(s.Key, "-") {
		return fmt.Errorf("invalid key format %q, expected PROJECT-NUMBER", s.Key)
	}
	return s.Staleness.Validate()
}

// PRSpec holds type-specific fields for a GitHub pull request.
type PRSpec struct {
	// Repo is the GitHub owner/repo slug. Required.
	// Example: "Azure/ARO-HCP"
	Repo string `json:"repo"`

	// Number is the PR number within the repository. Required.
	Number int `json:"number"`

	// Branch is the head branch name of the PR. Required.
	Branch string `json:"branch"`

	// BranchState indicates the state of the PR branch relative to its
	// target branch. Must be a valid BranchState value.
	// Empty (BranchStateUnknown) means not yet checked.
	BranchState BranchState `json:"branch_state,omitempty"`

	// SplitSurfaceID is the Supacode surface ID if a Claude split session
	// is currently open for this PR. Empty when no split is active.
	SplitSurfaceID string `json:"split_surface_id,omitempty"`
}

// Validate checks required fields, length constraints, and enum values.
func (s *PRSpec) Validate() error {
	if len(s.Repo) == 0 {
		return fmt.Errorf("repo is required")
	}
	if len(s.Repo) > 200 {
		return fmt.Errorf("repo: max 200 chars, got %d", len(s.Repo))
	}
	if !strings.Contains(s.Repo, "/") {
		return fmt.Errorf("invalid repo format %q, expected owner/repo", s.Repo)
	}
	if s.Number <= 0 {
		return fmt.Errorf("number must be positive, got %d", s.Number)
	}
	if len(s.Branch) == 0 {
		return fmt.Errorf("branch is required")
	}
	if len(s.Branch) > 500 {
		return fmt.Errorf("branch: max 500 chars, got %d", len(s.Branch))
	}
	if len(s.SplitSurfaceID) > 500 {
		return fmt.Errorf("split_surface_id: max 500 chars, got %d", len(s.SplitSurfaceID))
	}
	return s.BranchState.Validate()
}

// CheckSpec holds type-specific fields for a CI check run on a PR.
type CheckSpec struct {
	// Name is the CI check name as reported by GitHub, e.g. "e2e-test-suite".
	Name string `json:"name"`
}

// Validate checks required fields and length constraints.
func (s *CheckSpec) Validate() error {
	if len(s.Name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(s.Name) > 500 {
		return fmt.Errorf("name: max 500 chars, got %d", len(s.Name))
	}
	return nil
}

// ReviewSpec holds type-specific fields for a review comment on a PR.
type ReviewSpec struct {
	// File is the path of the file the comment is attached to.
	File string `json:"file"`

	// BodyHash is a truncated hash of the comment body, used by the
	// informer to detect content changes without storing full text.
	BodyHash string `json:"body_hash"`

	// Summary is an LLM-generated summary of the review comment,
	// capped at ~40 characters for TUI display.
	Summary string `json:"summary"`

	// SplitSurfaceID is the Supacode surface ID if a Claude split session
	// is currently open for this review comment. Empty when no split is active.
	SplitSurfaceID string `json:"split_surface_id,omitempty"`
}

// Validate checks required fields and length constraints.
func (s *ReviewSpec) Validate() error {
	if len(s.File) == 0 {
		return fmt.Errorf("file is required")
	}
	if len(s.File) > 4096 {
		return fmt.Errorf("file: max 4096 chars, got %d", len(s.File))
	}
	if len(s.BodyHash) > 64 {
		return fmt.Errorf("body_hash: max 64 chars, got %d", len(s.BodyHash))
	}
	if len(s.Summary) > 200 {
		return fmt.Errorf("summary: max 200 chars, got %d", len(s.Summary))
	}
	if len(s.SplitSurfaceID) > 500 {
		return fmt.Errorf("split_surface_id: max 500 chars, got %d", len(s.SplitSurfaceID))
	}
	return nil
}

// LocalSpec holds type-specific fields for a local worktree with no PR yet.
type LocalSpec struct {
	// WorktreeID is an opaque identifier from Supacode — never parse or decode.
	WorktreeID string `json:"worktree_id"`

	// Branch is the git branch name checked out in the worktree.
	Branch string `json:"branch"`

	// JiraKey is the Jira issue key extracted from the branch name via
	// regex, if one was found. Empty when no match.
	JiraKey string `json:"jira_key,omitempty"`
}

// Validate checks required fields and length constraints.
func (s *LocalSpec) Validate() error {
	if len(s.WorktreeID) == 0 {
		return fmt.Errorf("worktree_id is required")
	}
	if len(s.WorktreeID) > 4096 {
		return fmt.Errorf("worktree_id: max 4096 chars, got %d", len(s.WorktreeID))
	}
	if len(s.Branch) == 0 {
		return fmt.Errorf("branch is required")
	}
	if len(s.Branch) > 500 {
		return fmt.Errorf("branch: max 500 chars, got %d", len(s.Branch))
	}
	if len(s.JiraKey) > 50 {
		return fmt.Errorf("jira_key: max 50 chars, got %d", len(s.JiraKey))
	}
	return nil
}
