package workitem

type JiraSpec struct {
	Key   string `json:"key"`
	Stale bool   `json:"stale"`
}

type PRSpec struct {
	Repo           string `json:"repo"`
	Number         int    `json:"number"`
	Branch         string `json:"branch"`
	NeedsRebase    bool   `json:"needs_rebase"`
	SplitSurfaceID string `json:"split_surface_id,omitempty"`
}

type CheckSpec struct {
	Name string `json:"name"`
}

type ReviewSpec struct {
	File           string `json:"file"`
	BodyHash       string `json:"body_hash"`
	Summary        string `json:"summary"`
	SplitSurfaceID string `json:"split_surface_id,omitempty"`
}

type LocalSpec struct {
	// Opaque value from Supacode — never parse or decode.
	WorktreeID string `json:"worktree_id"`
	Branch     string `json:"branch"`
	JiraKey    string `json:"jira_key,omitempty"`
}
