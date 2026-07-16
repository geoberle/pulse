package api

func (w *Worktree) DeepCopy() *Worktree {
	out := *w
	return &out
}

func (w *Worktree) DeepCopyObject() Object {
	return w.DeepCopy()
}

func (p *PullRequest) DeepCopy() *PullRequest {
	out := *p
	return &out
}

func (p *PullRequest) DeepCopyObject() Object {
	return p.DeepCopy()
}

func (j *JiraTicket) DeepCopy() *JiraTicket {
	out := *j
	return &out
}

func (j *JiraTicket) DeepCopyObject() Object {
	return j.DeepCopy()
}

func (m *ManualLink) DeepCopy() *ManualLink {
	out := *m
	return &out
}

func (m *ManualLink) DeepCopyObject() Object {
	return m.DeepCopy()
}
