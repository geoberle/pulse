package api

import "k8s.io/apimachinery/pkg/runtime"

func (w *Worktree) DeepCopy() *Worktree {
	out := *w
	return &out
}

func (w *Worktree) DeepCopyObject() runtime.Object {
	return w.DeepCopy()
}

func (l *WorktreeList) DeepCopy() *WorktreeList {
	out := *l
	out.Items = make([]Worktree, len(l.Items))
	copy(out.Items, l.Items)
	return &out
}

func (l *WorktreeList) DeepCopyObject() runtime.Object {
	return l.DeepCopy()
}

func (p *PullRequest) DeepCopy() *PullRequest {
	out := *p
	return &out
}

func (p *PullRequest) DeepCopyObject() runtime.Object {
	return p.DeepCopy()
}

func (l *PullRequestList) DeepCopy() *PullRequestList {
	out := *l
	out.Items = make([]PullRequest, len(l.Items))
	copy(out.Items, l.Items)
	return &out
}

func (l *PullRequestList) DeepCopyObject() runtime.Object {
	return l.DeepCopy()
}

func (j *JiraTicket) DeepCopy() *JiraTicket {
	out := *j
	return &out
}

func (j *JiraTicket) DeepCopyObject() runtime.Object {
	return j.DeepCopy()
}

func (l *JiraTicketList) DeepCopy() *JiraTicketList {
	out := *l
	out.Items = make([]JiraTicket, len(l.Items))
	copy(out.Items, l.Items)
	return &out
}

func (l *JiraTicketList) DeepCopyObject() runtime.Object {
	return l.DeepCopy()
}

func (m *ManualLink) DeepCopy() *ManualLink {
	out := *m
	return &out
}

func (m *ManualLink) DeepCopyObject() runtime.Object {
	return m.DeepCopy()
}

func (l *ManualLinkList) DeepCopy() *ManualLinkList {
	out := *l
	out.Items = make([]ManualLink, len(l.Items))
	copy(out.Items, l.Items)
	return &out
}

func (l *ManualLinkList) DeepCopyObject() runtime.Object {
	return l.DeepCopy()
}
