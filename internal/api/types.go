package api

import (
	"fmt"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ObjectMeta struct {
	ResourceVersion   int64     `json:"resourceVersion"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
}

type WorktreeCommitState string

const (
	WorktreeCommitStateNone       WorktreeCommitState = ""
	WorktreeCommitStateHasCommits WorktreeCommitState = "HasCommits"
)

type Worktree struct {
	ObjectMeta
	Path        string              `json:"path"`
	Repo        string              `json:"repo"`
	Branch      string              `json:"branch"`
	CommitState WorktreeCommitState `json:"commitState"`
	LastSeen    time.Time           `json:"lastSeen"`
}

var (
	_ runtime.Object            = &Worktree{}
	_ metav1.ObjectMetaAccessor = &Worktree{}
)

func (w *Worktree) Key() string                      { return w.Path }
func (w *Worktree) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (w *Worktree) GetObjectMeta() metav1.Object {
	return &metav1.ObjectMeta{
		Name:            w.Path,
		ResourceVersion: strconv.FormatInt(w.ResourceVersion, 10),
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorktreeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Worktree `json:"items"`
}

var _ runtime.Object = &WorktreeList{}

func (l *WorktreeList) GetObjectKind() schema.ObjectKind { return &l.TypeMeta }

type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "open"
	PullRequestStatusClosed PullRequestStatus = "closed"
	PullRequestStatusMerged PullRequestStatus = "merged"
)

type CIStatus string

const (
	CIStatusPassing CIStatus = "passing"
	CIStatusFailing CIStatus = "failing"
	CIStatusPending CIStatus = "pending"
)

type ReviewStatus string

const (
	ReviewStatusApproved         ReviewStatus = "approved"
	ReviewStatusChangesRequested ReviewStatus = "changes_requested"
	ReviewStatusPending          ReviewStatus = "pending"
)

type PullRequest struct {
	ObjectMeta
	Repo               string            `json:"repo"`
	Number             int               `json:"number"`
	Branch             string            `json:"branch"`
	URL                string            `json:"url"`
	Status             PullRequestStatus `json:"status"`
	CIStatus           CIStatus          `json:"ciStatus"`
	ReviewStatus       ReviewStatus      `json:"reviewStatus"`
	UnresolvedComments int               `json:"unresolvedComments"`
	Author             string            `json:"author"`
	LastSeen           time.Time         `json:"lastSeen"`
}

var (
	_ runtime.Object            = &PullRequest{}
	_ metav1.ObjectMetaAccessor = &PullRequest{}
)

func (p *PullRequest) Key() string                      { return p.Repo + "#" + strconv.Itoa(p.Number) }
func (p *PullRequest) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (p *PullRequest) GetObjectMeta() metav1.Object {
	return &metav1.ObjectMeta{
		Name:            p.Key(),
		ResourceVersion: strconv.FormatInt(p.ResourceVersion, 10),
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PullRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PullRequest `json:"items"`
}

var _ runtime.Object = &PullRequestList{}

func (l *PullRequestList) GetObjectKind() schema.ObjectKind { return &l.TypeMeta }

type JiraIssueType string
type JiraStatus string

type JiraTicket struct {
	ObjectMeta
	TicketKey    string        `json:"key"`
	Summary      string        `json:"summary"`
	Description  string        `json:"description"`
	Status       JiraStatus    `json:"status"`
	IssueType    JiraIssueType `json:"issueType"`
	EpicKey      string        `json:"epicKey"`
	LastActivity time.Time     `json:"lastActivity"`
	LastSeen     time.Time     `json:"lastSeen"`
}

var (
	_ runtime.Object            = &JiraTicket{}
	_ metav1.ObjectMetaAccessor = &JiraTicket{}
)

func (j *JiraTicket) Key() string                      { return j.TicketKey }
func (j *JiraTicket) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (j *JiraTicket) GetObjectMeta() metav1.Object {
	return &metav1.ObjectMeta{
		Name:            j.TicketKey,
		ResourceVersion: strconv.FormatInt(j.ResourceVersion, 10),
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JiraTicketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JiraTicket `json:"items"`
}

var _ runtime.Object = &JiraTicketList{}

func (l *JiraTicketList) GetObjectKind() schema.ObjectKind { return &l.TypeMeta }

type ManualLinkSourceType string

const (
	ManualLinkSourceTypeWorktree    ManualLinkSourceType = "worktree"
	ManualLinkSourceTypePullRequest ManualLinkSourceType = "pullrequest"
)

type ManualLink struct {
	ObjectMeta
	SourceType ManualLinkSourceType `json:"sourceType"`
	SourceID   string               `json:"sourceID"`
	JiraKey    string               `json:"jiraKey"`
}

var (
	_ runtime.Object            = &ManualLink{}
	_ metav1.ObjectMetaAccessor = &ManualLink{}
)

func (m *ManualLink) Key() string                      { return string(m.SourceType) + "/" + m.SourceID }
func (m *ManualLink) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (m *ManualLink) GetObjectMeta() metav1.Object {
	return &metav1.ObjectMeta{
		Name:            m.Key(),
		ResourceVersion: strconv.FormatInt(m.ResourceVersion, 10),
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManualLinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManualLink `json:"items"`
}

var _ runtime.Object = &ManualLinkList{}

func (l *ManualLinkList) GetObjectKind() schema.ObjectKind { return &l.TypeMeta }

// Validation

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
