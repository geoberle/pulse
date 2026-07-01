package workitem

import (
	"fmt"
	"regexp"
	"strings"
)

var jiraKeyRegex = regexp.MustCompile(`^[A-Z][A-Z0-9]*-\d+$`)

// ValidateRepoFormat checks that repo is a valid "owner/repo" slug.
func ValidateRepoFormat(repo string) error {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return fmt.Errorf("invalid repo format %q, expected owner/repo", repo)
	}
	return nil
}

// ConditionType identifies a WorkItem condition (e.g. Stale, BranchOutdated).
type ConditionType string

const (
	ConditionStale          ConditionType = "Stale"
	ConditionBranchOutdated ConditionType = "BranchOutdated"
)

// ConditionReason explains why a condition has its current status.
type ConditionReason string

const (
	ReasonThresholdExceeded ConditionReason = "ThresholdExceeded"
	ReasonWithinThreshold   ConditionReason = "WithinThreshold"
	ReasonNeedsRebase       ConditionReason = "NeedsRebase"
	ReasonUpToDate          ConditionReason = "UpToDate"
)

// specFactories is the single source of truth for valid Kinds. Maps each
// Kind to a factory that creates an empty Spec for unmarshaling.
// Kind.Validate() and UnmarshalSpec both use this registry.
var specFactories = map[Kind]func() Spec{
	KindJira:   func() Spec { return &JiraSpec{} },
	KindPR:     func() Spec { return &PRSpec{} },
	KindCheck:  func() Spec { return &CheckSpec{} },
	KindReview: func() Spec { return &ReviewSpec{} },
	KindLocal:  func() Spec { return &LocalSpec{} },
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
	Key string `json:"key"`

	// Summary is the Jira issue summary (display text for TUI).
	Summary string `json:"summary"`
}

// Validate checks required fields, length constraints, and format.
func (s *JiraSpec) Validate() error {
	if len(s.Key) == 0 {
		return fmt.Errorf("key is required")
	}
	if len(s.Key) > 50 {
		return fmt.Errorf("key: max 50 chars, got %d", len(s.Key))
	}
	if !jiraKeyRegex.MatchString(s.Key) {
		return fmt.Errorf("invalid key format %q, expected PROJECT-NUMBER (e.g. ARO-123)", s.Key)
	}
	if len(s.Summary) == 0 {
		return fmt.Errorf("summary is required")
	}
	if len(s.Summary) > 1000 {
		return fmt.Errorf("summary: max 1000 chars, got %d", len(s.Summary))
	}
	return nil
}

// PRSpec holds type-specific fields for a GitHub pull request.
type PRSpec struct {
	// Repo is the GitHub owner/repo slug. Required.
	Repo string `json:"repo"`

	// Number is the PR number within the repository. Required.
	Number int `json:"number"`

	// Branch is the head branch name of the PR. Required.
	Branch string `json:"branch"`

	// Title is the PR title (display text for TUI). Required.
	Title string `json:"title"`

	// SplitSurfaceID is the Supacode surface ID if a Claude split session
	// is currently open for this PR. Empty when no split is active.
	SplitSurfaceID string `json:"split_surface_id,omitempty"`
}

// Validate checks required fields, length constraints, and format.
func (s *PRSpec) Validate() error {
	if len(s.Repo) == 0 {
		return fmt.Errorf("repo is required")
	}
	if len(s.Repo) > 200 {
		return fmt.Errorf("repo: max 200 chars, got %d", len(s.Repo))
	}
	if err := ValidateRepoFormat(s.Repo); err != nil {
		return err
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
	if len(s.Title) == 0 {
		return fmt.Errorf("title is required")
	}
	if len(s.Title) > 1000 {
		return fmt.Errorf("title: max 1000 chars, got %d", len(s.Title))
	}
	if len(s.SplitSurfaceID) > 500 {
		return fmt.Errorf("split_surface_id: max 500 chars, got %d", len(s.SplitSurfaceID))
	}
	return nil
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

func (s *JiraSpec) DeepCopySpec() Spec   { return s.DeepCopy() }
func (s *PRSpec) DeepCopySpec() Spec     { return s.DeepCopy() }
func (s *CheckSpec) DeepCopySpec() Spec  { return s.DeepCopy() }
func (s *ReviewSpec) DeepCopySpec() Spec { return s.DeepCopy() }
func (s *LocalSpec) DeepCopySpec() Spec  { return s.DeepCopy() }
