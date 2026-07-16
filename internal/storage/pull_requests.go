package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/geoberle/pulse/internal/api"
)

type pullRequestClient struct {
	db          *sql.DB
	broadcaster *watch.Broadcaster
}

func (c *pullRequestClient) Get(ctx context.Context, key string) (*api.PullRequest, error) {
	repo, number, err := parsePRKey(key)
	if err != nil {
		return nil, err
	}
	row := c.db.QueryRowContext(ctx,
		`SELECT repo, number, branch, url, status, ci_status, review_status,
		unresolved_comments, author, last_seen, resource_version, created_at
		FROM pull_requests WHERE repo = ? AND number = ?`, repo, number)
	return scanPullRequest(row)
}

func (c *pullRequestClient) List(ctx context.Context) ([]*api.PullRequest, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT repo, number, branch, url, status, ci_status, review_status,
		unresolved_comments, author, last_seen, resource_version, created_at
		FROM pull_requests ORDER BY repo, number`)
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
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate pull request: %w", err)
	}
	now := time.Now().UTC()
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO pull_requests (repo, number, branch, url, status, ci_status,
		review_status, unresolved_comments, author, last_seen, resource_version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)`,
		obj.Repo, obj.Number, obj.Branch, obj.URL, string(obj.Status), string(obj.CIStatus),
		string(obj.ReviewStatus), obj.UnresolvedComments, obj.Author,
		obj.LastSeen.UTC().Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create pull request %s#%d: %w", obj.Repo, obj.Number, err)
	}
	created, err := c.Get(ctx, obj.Key())
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
	res, err := c.db.ExecContext(ctx,
		`UPDATE pull_requests SET branch = ?, url = ?, status = ?, ci_status = ?,
		review_status = ?, unresolved_comments = ?, author = ?, last_seen = ?,
		resource_version = resource_version + 1
		WHERE repo = ? AND number = ? AND resource_version = ?`,
		obj.Branch, obj.URL, string(obj.Status), string(obj.CIStatus),
		string(obj.ReviewStatus), obj.UnresolvedComments, obj.Author,
		obj.LastSeen.UTC().Format(time.RFC3339Nano),
		obj.Repo, obj.Number, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update pull request %s#%d: %w", obj.Repo, obj.Number, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := c.Get(ctx, obj.Key()); err != nil {
			return nil, ErrNotFound
		}
		return nil, ErrConflict
	}
	updated, err := c.Get(ctx, obj.Key())
	if err != nil {
		return nil, err
	}
	_ = c.broadcaster.Action(watch.Modified, updated)
	return updated, nil
}

func (c *pullRequestClient) Delete(ctx context.Context, key string) error {
	obj, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	repo, number, err := parsePRKey(key)
	if err != nil {
		return err
	}
	res, err := c.db.ExecContext(ctx,
		"DELETE FROM pull_requests WHERE repo = ? AND number = ?", repo, number)
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

func parsePRKey(key string) (string, int, error) {
	idx := strings.LastIndex(key, "#")
	if idx < 0 {
		return "", 0, fmt.Errorf("invalid pull request key %q: expected repo#number", key)
	}
	repo := key[:idx]
	number, err := strconv.Atoi(key[idx+1:])
	if err != nil {
		return "", 0, fmt.Errorf("invalid pull request key %q: %w", key, err)
	}
	if len(repo) == 0 || number <= 0 {
		return "", 0, fmt.Errorf("invalid pull request key %q: empty repo or non-positive number", key)
	}
	return repo, number, nil
}

func scanPullRequest(row *sql.Row) (*api.PullRequest, error) {
	var pr api.PullRequest
	var status, ciStatus, reviewStatus, lastSeen, createdAt string
	err := row.Scan(&pr.Repo, &pr.Number, &pr.Branch, &pr.URL, &status,
		&ciStatus, &reviewStatus, &pr.UnresolvedComments, &pr.Author,
		&lastSeen, &pr.ResourceVersion, &createdAt)
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
	var parseErr error
	if pr.LastSeen, parseErr = time.Parse(time.RFC3339Nano, lastSeen); parseErr != nil {
		return nil, fmt.Errorf("parse pull request last_seen: %w", parseErr)
	}
	if pr.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse pull request created_at: %w", parseErr)
	}
	return &pr, nil
}

func scanPullRequestRow(rows *sql.Rows) (*api.PullRequest, error) {
	var pr api.PullRequest
	var status, ciStatus, reviewStatus, lastSeen, createdAt string
	err := rows.Scan(&pr.Repo, &pr.Number, &pr.Branch, &pr.URL, &status,
		&ciStatus, &reviewStatus, &pr.UnresolvedComments, &pr.Author,
		&lastSeen, &pr.ResourceVersion, &createdAt)
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
	var parseErr error
	if pr.LastSeen, parseErr = time.Parse(time.RFC3339Nano, lastSeen); parseErr != nil {
		return nil, fmt.Errorf("parse pull request last_seen: %w", parseErr)
	}
	if pr.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse pull request created_at: %w", parseErr)
	}
	return &pr, nil
}
