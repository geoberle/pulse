package storage

import (
	"context"
	"database/sql"
	"encoding/json"
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
		`SELECT name, repo, branch, commit_state, deletion_timestamp, finalizers,
		resource_version, created_at
		FROM worktrees WHERE name = ?`, key)
	return scanWorktree(row)
}

func (c *worktreeClient) List(ctx context.Context) ([]*api.Worktree, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT name, repo, branch, commit_state, deletion_timestamp, finalizers,
		resource_version, created_at
		FROM worktrees ORDER BY name`)
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
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO worktrees (name, repo, branch, commit_state, finalizers, resource_version, created_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)`,
		obj.Name, obj.Repo, obj.Branch, string(obj.CommitState),
		finalizersJSON, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create worktree %s: %w", obj.Name, err)
	}
	created, err := c.Get(ctx, obj.Name)
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
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	var deletionTS *string
	if obj.DeletionTimestamp != nil {
		s := obj.DeletionTimestamp.UTC().Format(time.RFC3339Nano)
		deletionTS = &s
	}
	res, err := c.db.ExecContext(ctx,
		`UPDATE worktrees SET repo = ?, branch = ?, commit_state = ?,
		deletion_timestamp = ?, finalizers = ?,
		resource_version = resource_version + 1
		WHERE name = ? AND resource_version = ?`,
		obj.Repo, obj.Branch, string(obj.CommitState),
		deletionTS, finalizersJSON,
		obj.Name, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update worktree %s: %w", obj.Name, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := c.Get(ctx, obj.Name); err != nil {
			return nil, ErrNotFound
		}
		return nil, ErrConflict
	}
	updated, err := c.Get(ctx, obj.Name)
	if err != nil {
		return nil, err
	}
	if updated.DeletionTimestamp != nil && len(updated.Finalizers) == 0 {
		if _, err := c.db.ExecContext(ctx, "DELETE FROM worktrees WHERE name = ?", obj.Name); err != nil {
			return nil, fmt.Errorf("garbage collect worktree %s: %w", obj.Name, err)
		}
		_ = c.broadcaster.Action(watch.Deleted, updated)
		return updated, nil
	}
	_ = c.broadcaster.Action(watch.Modified, updated)
	return updated, nil
}

func (c *worktreeClient) Delete(ctx context.Context, key string) error {
	for range 2 {
		obj, err := c.Get(ctx, key)
		if err != nil {
			return err
		}
		if obj.DeletionTimestamp != nil {
			return nil
		}
		if len(obj.Finalizers) > 0 {
			now := time.Now().UTC()
			res, err := c.db.ExecContext(ctx,
				`UPDATE worktrees SET deletion_timestamp = ?, resource_version = resource_version + 1
				WHERE name = ? AND resource_version = ?`,
				now.Format(time.RFC3339Nano), key, obj.ResourceVersion)
			if err != nil {
				return fmt.Errorf("soft delete worktree %s: %w", key, err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				continue
			}
			updated, err := c.Get(ctx, key)
			if err != nil {
				return err
			}
			_ = c.broadcaster.Action(watch.Modified, updated)
			return nil
		}
		res, err := c.db.ExecContext(ctx, "DELETE FROM worktrees WHERE name = ?", key)
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
	return ErrConflict
}

func scanWorktree(row *sql.Row) (*api.Worktree, error) {
	var wt api.Worktree
	var commitState, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := row.Scan(&wt.Name, &wt.Repo, &wt.Branch, &commitState,
		&deletionTS, &finalizersJSON, &wt.ResourceVersion, &createdAt)
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
	if err := parseScanFields(&wt.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("worktree %s: %w", wt.Name, err)
	}
	return &wt, nil
}

func scanWorktreeRow(rows *sql.Rows) (*api.Worktree, error) {
	var wt api.Worktree
	var commitState, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := rows.Scan(&wt.Name, &wt.Repo, &wt.Branch, &commitState,
		&deletionTS, &finalizersJSON, &wt.ResourceVersion, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan worktree row: %w", err)
	}
	wt.CommitState = api.WorktreeCommitState(commitState)
	if !wt.CommitState.Valid() {
		return nil, fmt.Errorf("invalid worktree commit_state %q", commitState)
	}
	if err := parseScanFields(&wt.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("worktree %s: %w", wt.Name, err)
	}
	return &wt, nil
}

func parseScanFields(meta *api.ObjectMeta, deletionTS sql.NullString, finalizersJSON, createdAt string) error {
	var err error
	if meta.CreationTimestamp, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return fmt.Errorf("parse created_at: %w", err)
	}
	if deletionTS.Valid {
		t, err := time.Parse(time.RFC3339Nano, deletionTS.String)
		if err != nil {
			return fmt.Errorf("parse deletion_timestamp: %w", err)
		}
		meta.DeletionTimestamp = &t
	}
	if err := json.Unmarshal([]byte(finalizersJSON), &meta.Finalizers); err != nil {
		return fmt.Errorf("parse finalizers: %w", err)
	}
	return nil
}

func marshalFinalizers(finalizers []string) string {
	if len(finalizers) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(finalizers)
	return string(b)
}
