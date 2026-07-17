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
	Name              string     `json:"name"`
	ResourceVersion   int64      `json:"resourceVersion"`
	CreationTimestamp time.Time  `json:"creationTimestamp"`
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"`
	Finalizers        []string   `json:"finalizers,omitempty"`
}

type WorktreeCommitState string

const (
	WorktreeCommitStateNone       WorktreeCommitState = ""
	WorktreeCommitStateHasCommits WorktreeCommitState = "HasCommits"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Worktree struct {
	ObjectMeta
	Repo        string              `json:"repo"`
	Branch      string              `json:"branch"`
	CommitState WorktreeCommitState `json:"commitState"`
}

var (
	_ runtime.Object            = &Worktree{}
	_ metav1.ObjectMetaAccessor = &Worktree{}
)

func (w *Worktree) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (w *Worktree) GetObjectMeta() metav1.Object {
	om := &metav1.ObjectMeta{
		Name:              w.Name,
		ResourceVersion:   strconv.FormatInt(w.ResourceVersion, 10),
		CreationTimestamp: metav1.NewTime(w.CreationTimestamp),
		Finalizers:        w.Finalizers,
	}
	if w.DeletionTimestamp != nil {
		om.DeletionTimestamp = &metav1.Time{Time: *w.DeletionTimestamp}
	}
	return om
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

func PullRequestName(repo string, number int) string {
	return repo + "#" + strconv.Itoa(number)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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
}

var (
	_ runtime.Object            = &PullRequest{}
	_ metav1.ObjectMetaAccessor = &PullRequest{}
)

func (p *PullRequest) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (p *PullRequest) GetObjectMeta() metav1.Object {
	om := &metav1.ObjectMeta{
		Name:              p.Name,
		ResourceVersion:   strconv.FormatInt(p.ResourceVersion, 10),
		CreationTimestamp: metav1.NewTime(p.CreationTimestamp),
		Finalizers:        p.Finalizers,
	}
	if p.DeletionTimestamp != nil {
		om.DeletionTimestamp = &metav1.Time{Time: *p.DeletionTimestamp}
	}
	return om
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JiraTicket struct {
	ObjectMeta
	Summary      string        `json:"summary"`
	Description  string        `json:"description"`
	Status       JiraStatus    `json:"status"`
	IssueType    JiraIssueType `json:"issueType"`
	EpicKey      string        `json:"epicKey"`
	LastActivity time.Time     `json:"lastActivity"`
}

var (
	_ runtime.Object            = &JiraTicket{}
	_ metav1.ObjectMetaAccessor = &JiraTicket{}
)

func (j *JiraTicket) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (j *JiraTicket) GetObjectMeta() metav1.Object {
	om := &metav1.ObjectMeta{
		Name:              j.Name,
		ResourceVersion:   strconv.FormatInt(j.ResourceVersion, 10),
		CreationTimestamp: metav1.NewTime(j.CreationTimestamp),
		Finalizers:        j.Finalizers,
	}
	if j.DeletionTimestamp != nil {
		om.DeletionTimestamp = &metav1.Time{Time: *j.DeletionTimestamp}
	}
	return om
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

func ManualLinkName(sourceType ManualLinkSourceType, sourceID string) string {
	return string(sourceType) + "/" + sourceID
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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

func (m *ManualLink) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (m *ManualLink) GetObjectMeta() metav1.Object {
	om := &metav1.ObjectMeta{
		Name:              m.Name,
		ResourceVersion:   strconv.FormatInt(m.ResourceVersion, 10),
		CreationTimestamp: metav1.NewTime(m.CreationTimestamp),
		Finalizers:        m.Finalizers,
	}
	if m.DeletionTimestamp != nil {
		om.DeletionTimestamp = &metav1.Time{Time: *m.DeletionTimestamp}
	}
	return om
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

func (o *ObjectMeta) Validate() error {
	seen := make(map[string]struct{}, len(o.Finalizers))
	for _, f := range o.Finalizers {
		if len(f) == 0 {
			return fmt.Errorf("empty finalizer")
		}
		if _, ok := seen[f]; ok {
			return fmt.Errorf("duplicate finalizer %q", f)
		}
		seen[f] = struct{}{}
	}
	return nil
}

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
	if err := w.ObjectMeta.Validate(); err != nil {
		return err
	}
	if len(w.Name) == 0 {
		return fmt.Errorf("name is required")
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
	if err := p.ObjectMeta.Validate(); err != nil {
		return err
	}
	if len(p.Name) == 0 {
		return fmt.Errorf("name is required")
	}
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
	if err := j.ObjectMeta.Validate(); err != nil {
		return err
	}
	if len(j.Name) == 0 {
		return fmt.Errorf("name is required")
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
	if err := m.ObjectMeta.Validate(); err != nil {
		return err
	}
	if len(m.Name) == 0 {
		return fmt.Errorf("name is required")
	}
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
