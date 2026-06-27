package supacode

// FocusedState indicates whether an item currently has focus.
// The zero value (empty string) means unfocused.
type FocusedState string

const (
	FocusedStateActive   FocusedState = "Active"
	FocusedStateInactive FocusedState = ""
)

// RunningState indicates whether a script is currently executing.
// The zero value (empty string) means not running.
type RunningState string

const (
	RunningStateActive   RunningState = "Active"
	RunningStateInactive RunningState = ""
)

// Repo represents a Supacode-managed repository.
type Repo struct {
	ID string `json:"id"`
}

// Worktree represents a git worktree managed by Supacode.
type Worktree struct {
	ID      string       `json:"id"`
	Focused FocusedState `json:"focused,omitempty"`
}

// Tab represents a terminal tab within a Supacode worktree.
type Tab struct {
	ID      string       `json:"id"`
	Focused FocusedState `json:"focused,omitempty"`
}

// Surface represents a terminal pane within a Supacode tab.
type Surface struct {
	ID      string       `json:"id"`
	Focused FocusedState `json:"focused,omitempty"`
}

// ScriptKind identifies the type of a Supacode script. Values are opaque —
// the set is defined by the Supacode protocol and may change.
type ScriptKind string

// Script represents a runnable script within a Supacode worktree.
type Script struct {
	ID          string       `json:"id"`
	Kind        ScriptKind   `json:"kind"`
	Name        string       `json:"name"`
	DisplayName string       `json:"displayName"`
	Running     RunningState `json:"running,omitempty"`
}

// queryRequest is the JSON envelope sent to Supacode for read operations.
type queryRequest struct {
	Query      string `json:"query"`
	WorktreeID string `json:"worktreeID,omitempty"`
	TabID      string `json:"tabID,omitempty"`
}

// commandRequest is the JSON envelope sent to Supacode for mutation operations.
type commandRequest struct {
	Deeplink string `json:"deeplink"`
}

// response is the JSON envelope returned by Supacode for all operations.
type response struct {
	OK   bool      `json:"ok"`
	Data []rawItem `json:"data,omitempty"`
	Err  string    `json:"error,omitempty"`
}

// rawItem holds the untyped string values from the protocol. All response
// values are strings — typed accessors convert them.
type rawItem struct {
	ID          string `json:"id"`
	Focused     string `json:"focused"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Running     string `json:"running"`
}

// Tolerate-on-read: only "1" is active; unknown values default to inactive.
func parseFocusedState(s string) FocusedState {
	if s == "1" {
		return FocusedStateActive
	}
	return FocusedStateInactive
}

// Tolerate-on-read: only "1" is active; unknown values default to inactive.
func parseRunningState(s string) RunningState {
	if s == "1" {
		return RunningStateActive
	}
	return RunningStateInactive
}

func (r rawItem) toRepo() Repo { return Repo{ID: r.ID} }
func (r rawItem) toWorktree() Worktree {
	return Worktree{ID: r.ID, Focused: parseFocusedState(r.Focused)}
}
func (r rawItem) toTab() Tab         { return Tab{ID: r.ID, Focused: parseFocusedState(r.Focused)} }
func (r rawItem) toSurface() Surface { return Surface{ID: r.ID, Focused: parseFocusedState(r.Focused)} }
func (r rawItem) toScript() Script {
	return Script{
		ID:          r.ID,
		Kind:        ScriptKind(r.Kind),
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Running:     parseRunningState(r.Running),
	}
}
