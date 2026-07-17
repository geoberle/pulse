package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/geoberle/pulse/internal/api"
)

type pullRequestClient struct {
	db          *sql.DB
	broadcaster *watch.Broadcaster
}

func (c *pullRequestClient) Get(ctx context.Context, key string) (*api.PullRequest, error) {
	row := c.db.QueryRowContext(ctx,
		`SELECT name, repo, number, branch, url, status, ci_status, review_status,
		unresolved_comments, author, deletion_timestamp, finalizers,
		resource_version, created_at
		FROM pull_requests WHERE name = ?`, key)
	return scanPullRequest(row)
}

func (c *pullRequestClient) List(ctx context.Context) ([]*api.PullRequest, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT name, repo, number, branch, url, status, ci_status, review_status,
		unresolved_comments, author, deletion_timestamp, finalizers,
		resource_version, created_at
		FROM pull_requests ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list pull requests: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*api.PullRequest
	for rows.Next() {
		pr, err := scanPullRequestRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

func (c *pullRequestClient) Create(ctx context.Context, obj *api.PullRequest) (*api.PullRequest, error) {
	obj.Name = api.PullRequestName(obj.Repo, obj.Number)
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate pull request: %w", err)
	}
	now := time.Now().UTC()
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO pull_requests (name, repo, number, branch, url, status, ci_status,
		review_status, unresolved_comments, author, finalizers, resource_version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)`,
		obj.Name, obj.Repo, obj.Number, obj.Branch, obj.URL, string(obj.Status), string(obj.CIStatus),
		string(obj.ReviewStatus), obj.UnresolvedComments, obj.Author,
		finalizersJSON, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create pull request %s: %w", obj.Name, err)
	}
	created, err := c.Get(ctx, obj.Name)
	if err != nil {
		return nil, err
	}
	_ = c.broadcaster.Action(watch.Added, created)
	return created, nil
}

func (c *pullRequestClient) Update(ctx context.Context, obj *api.PullRequest) (*api.PullRequest, error) {
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate pull request: %w", err)
	}
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	var deletionTS *string
	if obj.DeletionTimestamp != nil {
		s := obj.DeletionTimestamp.UTC().Format(time.RFC3339Nano)
		deletionTS = &s
	}
	res, err := c.db.ExecContext(ctx,
		`UPDATE pull_requests SET repo = ?, number = ?, branch = ?, url = ?, status = ?,
		ci_status = ?, review_status = ?, unresolved_comments = ?, author = ?,
		deletion_timestamp = ?, finalizers = ?,
		resource_version = resource_version + 1
		WHERE name = ? AND resource_version = ?`,
		obj.Repo, obj.Number, obj.Branch, obj.URL, string(obj.Status), string(obj.CIStatus),
		string(obj.ReviewStatus), obj.UnresolvedComments, obj.Author,
		deletionTS, finalizersJSON,
		obj.Name, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update pull request %s: %w", obj.Name, err)
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
		if _, err := c.db.ExecContext(ctx, "DELETE FROM pull_requests WHERE name = ?", obj.Name); err != nil {
			return nil, fmt.Errorf("garbage collect pull request %s: %w", obj.Name, err)
		}
		_ = c.broadcaster.Action(watch.Deleted, updated)
		return updated, nil
	}
	_ = c.broadcaster.Action(watch.Modified, updated)
	return updated, nil
}

func (c *pullRequestClient) Delete(ctx context.Context, key string) error {
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
				`UPDATE pull_requests SET deletion_timestamp = ?, resource_version = resource_version + 1
				WHERE name = ? AND resource_version = ?`,
				now.Format(time.RFC3339Nano), key, obj.ResourceVersion)
			if err != nil {
				return fmt.Errorf("soft delete pull request %s: %w", key, err)
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
		res, err := c.db.ExecContext(ctx, "DELETE FROM pull_requests WHERE name = ?", key)
		if err != nil {
			return fmt.Errorf("delete pull request %s: %w", key, err)
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

func scanPullRequest(row *sql.Row) (*api.PullRequest, error) {
	var pr api.PullRequest
	var status, ciStatus, reviewStatus, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := row.Scan(&pr.Name, &pr.Repo, &pr.Number, &pr.Branch, &pr.URL, &status,
		&ciStatus, &reviewStatus, &pr.UnresolvedComments, &pr.Author,
		&deletionTS, &finalizersJSON, &pr.ResourceVersion, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan pull request: %w", err)
	}
	pr.Status = api.PullRequestStatus(status)
	if !pr.Status.Valid() {
		return nil, fmt.Errorf("invalid pull request status %q", status)
	}
	pr.CIStatus = api.CIStatus(ciStatus)
	if !pr.CIStatus.Valid() {
		return nil, fmt.Errorf("invalid pull request ci_status %q", ciStatus)
	}
	pr.ReviewStatus = api.ReviewStatus(reviewStatus)
	if !pr.ReviewStatus.Valid() {
		return nil, fmt.Errorf("invalid pull request review_status %q", reviewStatus)
	}
	if err := parseScanFields(&pr.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("pull request %s: %w", pr.Name, err)
	}
	return &pr, nil
}

func scanPullRequestRow(rows *sql.Rows) (*api.PullRequest, error) {
	var pr api.PullRequest
	var status, ciStatus, reviewStatus, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := rows.Scan(&pr.Name, &pr.Repo, &pr.Number, &pr.Branch, &pr.URL, &status,
		&ciStatus, &reviewStatus, &pr.UnresolvedComments, &pr.Author,
		&deletionTS, &finalizersJSON, &pr.ResourceVersion, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan pull request row: %w", err)
	}
	pr.Status = api.PullRequestStatus(status)
	if !pr.Status.Valid() {
		return nil, fmt.Errorf("invalid pull request status %q", status)
	}
	pr.CIStatus = api.CIStatus(ciStatus)
	if !pr.CIStatus.Valid() {
		return nil, fmt.Errorf("invalid pull request ci_status %q", ciStatus)
	}
	pr.ReviewStatus = api.ReviewStatus(reviewStatus)
	if !pr.ReviewStatus.Valid() {
		return nil, fmt.Errorf("invalid pull request review_status %q", reviewStatus)
	}
	if err := parseScanFields(&pr.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("pull request %s: %w", pr.Name, err)
	}
	return &pr, nil
}
