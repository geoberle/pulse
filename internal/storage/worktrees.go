package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/geoberle/pulse/internal/api"
)

type worktreeClient struct {
	db          *sql.DB
	broadcaster *watch.Broadcaster
}

func (c *worktreeClient) Get(ctx context.Context, key string) (*api.Worktree, error) {
	row := c.db.QueryRowContext(ctx,
		`SELECT path, repo, branch, commit_state, last_seen, resource_version, created_at
		FROM worktrees WHERE path = ?`, key)
	return scanWorktree(row)
}

func (c *worktreeClient) List(ctx context.Context) ([]*api.Worktree, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT path, repo, branch, commit_state, last_seen, resource_version, created_at
		FROM worktrees ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*api.Worktree
	for rows.Next() {
		wt, err := scanWorktreeRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wt)
	}
	return out, rows.Err()
}

func (c *worktreeClient) Create(ctx context.Context, obj *api.Worktree) (*api.Worktree, error) {
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate worktree: %w", err)
	}
	now := time.Now().UTC()
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO worktrees (path, repo, branch, commit_state, last_seen, resource_version, created_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)`,
		obj.Path, obj.Repo, obj.Branch, string(obj.CommitState),
		obj.LastSeen.UTC().Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create worktree %s: %w", obj.Path, err)
	}
	created, err := c.Get(ctx, obj.Path)
	if err != nil {
		return nil, err
	}
	_ = c.broadcaster.Action(watch.Added, created)
	return created, nil
}

func (c *worktreeClient) Update(ctx context.Context, obj *api.Worktree) (*api.Worktree, error) {
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate worktree: %w", err)
	}
	res, err := c.db.ExecContext(ctx,
		`UPDATE worktrees SET repo = ?, branch = ?, commit_state = ?, last_seen = ?,
		resource_version = resource_version + 1
		WHERE path = ? AND resource_version = ?`,
		obj.Repo, obj.Branch, string(obj.CommitState),
		obj.LastSeen.UTC().Format(time.RFC3339Nano),
		obj.Path, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update worktree %s: %w", obj.Path, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := c.Get(ctx, obj.Path); err != nil {
			return nil, ErrNotFound
		}
		return nil, ErrConflict
	}
	updated, err := c.Get(ctx, obj.Path)
	if err != nil {
		return nil, err
	}
	_ = c.broadcaster.Action(watch.Modified, updated)
	return updated, nil
}

func (c *worktreeClient) Delete(ctx context.Context, key string) error {
	obj, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	res, err := c.db.ExecContext(ctx, "DELETE FROM worktrees WHERE path = ?", key)
	if err != nil {
		return fmt.Errorf("delete worktree %s: %w", key, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	_ = c.broadcaster.Action(watch.Deleted, obj)
	return nil
}

func scanWorktree(row *sql.Row) (*api.Worktree, error) {
	var wt api.Worktree
	var commitState, lastSeen, createdAt string
	err := row.Scan(&wt.Path, &wt.Repo, &wt.Branch, &commitState,
		&lastSeen, &wt.ResourceVersion, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan worktree: %w", err)
	}
	wt.CommitState = api.WorktreeCommitState(commitState)
	if !wt.CommitState.Valid() {
		return nil, fmt.Errorf("invalid worktree commit_state %q", commitState)
	}
	var parseErr error
	if wt.LastSeen, parseErr = time.Parse(time.RFC3339Nano, lastSeen); parseErr != nil {
		return nil, fmt.Errorf("parse worktree last_seen: %w", parseErr)
	}
	if wt.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse worktree created_at: %w", parseErr)
	}
	return &wt, nil
}

func scanWorktreeRow(rows *sql.Rows) (*api.Worktree, error) {
	var wt api.Worktree
	var commitState, lastSeen, createdAt string
	err := rows.Scan(&wt.Path, &wt.Repo, &wt.Branch, &commitState,
		&lastSeen, &wt.ResourceVersion, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan worktree row: %w", err)
	}
	wt.CommitState = api.WorktreeCommitState(commitState)
	if !wt.CommitState.Valid() {
		return nil, fmt.Errorf("invalid worktree commit_state %q", commitState)
	}
	var parseErr error
	if wt.LastSeen, parseErr = time.Parse(time.RFC3339Nano, lastSeen); parseErr != nil {
		return nil, fmt.Errorf("parse worktree last_seen: %w", parseErr)
	}
	if wt.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse worktree created_at: %w", parseErr)
	}
	return &wt, nil
}
