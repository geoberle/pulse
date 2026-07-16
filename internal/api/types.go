package api

import (
	"fmt"
	"strconv"
	"time"
)

// ObjectMeta contains metadata common to all stored objects.
type ObjectMeta struct {
	// ResourceVersion is an opaque counter incremented on every update,
	// used for optimistic concurrency control.
	ResourceVersion int64 `json:"resourceVersion"`
	// CreationTimestamp records when the object was first persisted.
	CreationTimestamp time.Time `json:"creationTimestamp"`
}

func (o *ObjectMeta) GetObjectMeta() *ObjectMeta {
	return o
}

// Object is the interface implemented by all API types that can be
// stored, listed, and watched via informers.
type Object interface {
	GetObjectMeta() *ObjectMeta
	Key() string
	DeepCopyObject() Object
}

// WorktreeCommitState indicates whether a worktree has local commits.
type WorktreeCommitState string

const (
	// WorktreeCommitStateNone means no local commits exist.
	WorktreeCommitStateNone WorktreeCommitState = ""
	// WorktreeCommitStateHasCommits means the worktree has unpushed commits.
	WorktreeCommitStateHasCommits WorktreeCommitState = "HasCommits"
)

// Worktree represents a local git worktree being tracked.
type Worktree struct {
	ObjectMeta
	// Path is the absolute filesystem path to the worktree root. Used as the primary key.
	Path string `json:"path"`
	// Repo is the repository identifier in "org/name" format, e.g. "Azure/ARO-HCP".
	Repo string `json:"repo"`
	// Branch is the git branch checked out in this worktree.
	Branch string `json:"branch"`
	// CommitState indicates whether local commits exist in this worktree.
	CommitState WorktreeCommitState `json:"commitState"`
	// LastSeen is the last time this worktree was observed on disk.
	LastSeen time.Time `json:"lastSeen"`
}

func (w *Worktree) Key() string { return w.Path }

// PullRequestStatus represents the lifecycle state of a pull request.
type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "open"
	PullRequestStatusClosed PullRequestStatus = "closed"
	PullRequestStatusMerged PullRequestStatus = "merged"
)

// CIStatus represents the continuous integration check result.
type CIStatus string

const (
	CIStatusPassing CIStatus = "passing"
	CIStatusFailing CIStatus = "failing"
	CIStatusPending CIStatus = "pending"
)

// ReviewStatus represents the code review state of a pull request.
type ReviewStatus string

const (
	ReviewStatusApproved         ReviewStatus = "approved"
	ReviewStatusChangesRequested ReviewStatus = "changes_requested"
	ReviewStatusPending          ReviewStatus = "pending"
)

// PullRequest represents a GitHub pull request being tracked.
type PullRequest struct {
	ObjectMeta
	// Repo is the repository identifier in "org/name" format.
	Repo string `json:"repo"`
	// Number is the PR number within the repository.
	Number int `json:"number"`
	// Branch is the source branch of this pull request.
	Branch string `json:"branch"`
	// URL is the web URL of the pull request.
	URL string `json:"url"`
	// Status is the lifecycle state: "open", "closed", or "merged".
	Status PullRequestStatus `json:"status"`
	// CIStatus is the aggregate CI check result: "passing", "failing", or "pending".
	CIStatus CIStatus `json:"ciStatus"`
	// ReviewStatus is the code review state: "approved", "changes_requested", or "pending".
	ReviewStatus ReviewStatus `json:"reviewStatus"`
	// UnresolvedComments is the count of unresolved review comments.
	UnresolvedComments int `json:"unresolvedComments"`
	// Author is the GitHub username of the PR author.
	Author string `json:"author"`
	// LastSeen is the last time this PR was observed via the GitHub API.
	LastSeen time.Time `json:"lastSeen"`
}

func (p *PullRequest) Key() string { return p.Repo + "#" + strconv.Itoa(p.Number) }

// JiraIssueType is a typed string for Jira issue types.
// Values are defined by the Jira project configuration (e.g. "Story", "Bug", "Task", "Epic").
type JiraIssueType string

// JiraStatus is a typed string for Jira workflow statuses.
// Values are defined by the Jira project configuration (e.g. "To Do", "In Progress", "Done").
type JiraStatus string

// JiraTicket represents a Jira issue being tracked.
type JiraTicket struct {
	ObjectMeta
	// TicketKey is the Jira issue key, e.g. "ARO-12345". Used as the primary key.
	TicketKey string `json:"key"`
	// Summary is the issue title/summary.
	Summary string `json:"summary"`
	// Description is the issue description body.
	Description string `json:"description"`
	// Status is the Jira workflow status, e.g. "In Progress", "Done".
	Status JiraStatus `json:"status"`
	// IssueType is the Jira issue type, e.g. "Story", "Bug".
	IssueType JiraIssueType `json:"issueType"`
	// EpicKey is the parent epic's Jira key, or empty if unlinked.
	EpicKey string `json:"epicKey"`
	// LastActivity is the timestamp of the most recent update in Jira.
	LastActivity time.Time `json:"lastActivity"`
	// LastSeen is the last time this ticket was observed via the Jira API.
	LastSeen time.Time `json:"lastSeen"`
}

func (j *JiraTicket) Key() string { return j.TicketKey }

// ManualLinkSourceType identifies the kind of source entity in a manual link.
type ManualLinkSourceType string

const (
	ManualLinkSourceTypeWorktree    ManualLinkSourceType = "worktree"
	ManualLinkSourceTypePullRequest ManualLinkSourceType = "pullrequest"
)

// ManualLink represents a user-created association between a source entity
// (worktree or pull request) and a Jira ticket.
type ManualLink struct {
	ObjectMeta
	// SourceType identifies the kind of source: "worktree" or "pullrequest".
	SourceType ManualLinkSourceType `json:"sourceType"`
	// SourceID is the identifier of the source entity (worktree path or PR key).
	SourceID string `json:"sourceID"`
	// JiraKey is the Jira issue key this source is linked to.
	JiraKey string `json:"jiraKey"`
}

func (m *ManualLink) Key() string { return string(m.SourceType) + "/" + m.SourceID }

func (s WorktreeCommitState) Valid() bool {
	switch s {
	case WorktreeCommitStateNone, WorktreeCommitStateHasCommits:
		return true
	}
	return false
}

func (s PullRequestStatus) Valid() bool {
	switch s {
	case PullRequestStatusOpen, PullRequestStatusClosed, PullRequestStatusMerged:
		return true
	}
	return false
}

func (s CIStatus) Valid() bool {
	switch s {
	case CIStatusPassing, CIStatusFailing, CIStatusPending:
		return true
	}
	return false
}

func (s ReviewStatus) Valid() bool {
	switch s {
	case ReviewStatusApproved, ReviewStatusChangesRequested, ReviewStatusPending:
		return true
	}
	return false
}

func (s ManualLinkSourceType) Valid() bool {
	switch s {
	case ManualLinkSourceTypeWorktree, ManualLinkSourceTypePullRequest:
		return true
	}
	return false
}

// JiraStatus and JiraIssueType are open enums defined by Jira project
// configuration. We validate non-empty only — the set of valid values
// is not known at compile time.

func (s JiraStatus) Valid() bool    { return len(s) > 0 }
func (s JiraIssueType) Valid() bool { return len(s) > 0 }

func (w *Worktree) Validate() error {
	if len(w.Path) == 0 {
		return fmt.Errorf("path is required")
	}
	if len(w.Repo) == 0 {
		return fmt.Errorf("repo is required")
	}
	if len(w.Branch) == 0 {
		return fmt.Errorf("branch is required")
	}
	if !w.CommitState.Valid() {
		return fmt.Errorf("invalid commit state %q", w.CommitState)
	}
	return nil
}

func (p *PullRequest) Validate() error {
	if len(p.Repo) == 0 {
		return fmt.Errorf("repo is required")
	}
	if p.Number <= 0 {
		return fmt.Errorf("number must be positive")
	}
	if !p.Status.Valid() {
		return fmt.Errorf("invalid status %q", p.Status)
	}
	if !p.CIStatus.Valid() {
		return fmt.Errorf("invalid ci status %q", p.CIStatus)
	}
	if !p.ReviewStatus.Valid() {
		return fmt.Errorf("invalid review status %q", p.ReviewStatus)
	}
	return nil
}

func (j *JiraTicket) Validate() error {
	if len(j.TicketKey) == 0 {
		return fmt.Errorf("ticket key is required")
	}
	if !j.Status.Valid() {
		return fmt.Errorf("status is required")
	}
	if !j.IssueType.Valid() {
		return fmt.Errorf("issue type is required")
	}
	return nil
}

func (m *ManualLink) Validate() error {
	if !m.SourceType.Valid() {
		return fmt.Errorf("invalid source type %q", m.SourceType)
	}
	if len(m.SourceID) == 0 {
		return fmt.Errorf("source ID is required")
	}
	if len(m.JiraKey) == 0 {
		return fmt.Errorf("jira key is required")
	}
	return nil
}
