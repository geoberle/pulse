package supacode

// Client provides typed query methods for the Supacode Unix socket protocol.
type Client struct {
	socketPath string
}

// NewClient creates a client that talks to the given socket path.
func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// ListRepos returns all repositories known to Supacode.
func (c *Client) ListRepos() ([]Repo, error) {
	resp, err := doQuery(c.socketPath, queryRequest{Query: "repos"})
	if err != nil {
		return nil, err
	}
	repos := make([]Repo, len(resp.Data))
	for i, item := range resp.Data {
		repos[i] = item.toRepo()
	}
	return repos, nil
}

// ListWorktrees returns all worktrees across all repos.
func (c *Client) ListWorktrees() ([]Worktree, error) {
	resp, err := doQuery(c.socketPath, queryRequest{Query: "worktrees"})
	if err != nil {
		return nil, err
	}
	wts := make([]Worktree, len(resp.Data))
	for i, item := range resp.Data {
		wts[i] = item.toWorktree()
	}
	return wts, nil
}

// ListTabs returns all tabs in a worktree.
func (c *Client) ListTabs(worktreeID string) ([]Tab, error) {
	resp, err := doQuery(c.socketPath, queryRequest{Query: "tabs", WorktreeID: worktreeID})
	if err != nil {
		return nil, err
	}
	tabs := make([]Tab, len(resp.Data))
	for i, item := range resp.Data {
		tabs[i] = item.toTab()
	}
	return tabs, nil
}

// ListSurfaces returns all surfaces (panes) in a tab.
func (c *Client) ListSurfaces(worktreeID, tabID string) ([]Surface, error) {
	resp, err := doQuery(c.socketPath, queryRequest{
		Query:      "surfaces",
		WorktreeID: worktreeID,
		TabID:      tabID,
	})
	if err != nil {
		return nil, err
	}
	surfaces := make([]Surface, len(resp.Data))
	for i, item := range resp.Data {
		surfaces[i] = item.toSurface()
	}
	return surfaces, nil
}

// ListScripts returns all scripts in a worktree.
func (c *Client) ListScripts(worktreeID string) ([]Script, error) {
	resp, err := doQuery(c.socketPath, queryRequest{Query: "scripts", WorktreeID: worktreeID})
	if err != nil {
		return nil, err
	}
	scripts := make([]Script, len(resp.Data))
	for i, item := range resp.Data {
		scripts[i] = item.toScript()
	}
	return scripts, nil
}
