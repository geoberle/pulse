package workitem

// StalenessState represents whether a Jira issue has gone stale based on
// the configured threshold. Empty string means staleness has not been
// evaluated yet.
type StalenessState string

const (
	StalenessUnknown StalenessState = ""
	StalenessActive  StalenessState = "Active"
	StalenessStale   StalenessState = "Stale"
)

// BranchState represents the state of a PR branch relative to its target.
// Empty string means branch state has not been checked.
type BranchState string

const (
	BranchStateUnknown     BranchState = ""
	BranchStateUpToDate    BranchState = "UpToDate"
	BranchStateNeedsRebase BranchState = "NeedsRebase"
)

// JiraSpec holds type-specific fields for a Jira work item (Story or Bug).
type JiraSpec struct {
	// Key is the Jira issue key, e.g. "ARO-12345".
	Key string `json:"key"`

	// Staleness indicates whether the issue has been inactive beyond the
	// configured stale_threshold_days. Empty means not yet evaluated.
	Staleness StalenessState `json:"staleness,omitempty"`
}

// PRSpec holds type-specific fields for a GitHub pull request.
type PRSpec struct {
	// Repo is the GitHub owner/repo slug, e.g. "Azure/ARO-HCP".
	Repo string `json:"repo"`

	// Number is the PR number within the repository.
	Number int `json:"number"`

	// Branch is the head branch name of the PR.
	Branch string `json:"branch"`

	// BranchState indicates the state of the PR branch relative to its
	// target branch (e.g. up-to-date, needs rebase). Empty means not yet checked.
	BranchState BranchState `json:"branch_state,omitempty"`

	// SplitSurfaceID is the Supacode surface ID if a Claude split session
	// is currently open for this PR. Empty when no split is active.
	SplitSurfaceID string `json:"split_surface_id,omitempty"`
}

// CheckSpec holds type-specific fields for a CI check run on a PR.
type CheckSpec struct {
	// Name is the CI check name as reported by GitHub, e.g. "e2e-test-suite".
	Name string `json:"name"`
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
