package supacode

// Repo represents a Supacode-managed repository.
type Repo struct {
	ID string `json:"id"`
}

// Worktree represents a git worktree managed by Supacode.
type Worktree struct {
	ID      string `json:"id"`
	Focused bool   `json:"focused"`
}

// Tab represents a terminal tab within a Supacode worktree.
type Tab struct {
	ID      string `json:"id"`
	Focused bool   `json:"focused"`
}

// Surface represents a terminal pane within a Supacode tab.
type Surface struct {
	ID      string `json:"id"`
	Focused bool   `json:"focused"`
}

// ScriptKind identifies the type of a Supacode script.
type ScriptKind string

// Script represents a runnable script within a Supacode worktree.
type Script struct {
	ID          string     `json:"id"`
	Kind        ScriptKind `json:"kind"`
	Name        string     `json:"name"`
	DisplayName string     `json:"displayName"`
	Running     bool       `json:"running"`
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
	OK   bool              `json:"ok"`
	Data []rawItem         `json:"data,omitempty"`
	Err  string            `json:"error,omitempty"`
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

func (r rawItem) isFocused() bool  { return r.Focused == "1" }
func (r rawItem) isRunning() bool  { return r.Running == "1" }

func (r rawItem) toRepo() Repo         { return Repo{ID: r.ID} }
func (r rawItem) toWorktree() Worktree  { return Worktree{ID: r.ID, Focused: r.isFocused()} }
func (r rawItem) toTab() Tab            { return Tab{ID: r.ID, Focused: r.isFocused()} }
func (r rawItem) toSurface() Surface    { return Surface{ID: r.ID, Focused: r.isFocused()} }
func (r rawItem) toScript() Script {
	return Script{
		ID:          r.ID,
		Kind:        ScriptKind(r.Kind),
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Running:     r.isRunning(),
	}
}
