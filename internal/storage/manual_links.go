package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/geoberle/pulse/internal/api"
)

type manualLinkClient struct {
	db          *sql.DB
	broadcaster *watch.Broadcaster
}

func (c *manualLinkClient) Get(ctx context.Context, key string) (*api.ManualLink, error) {
	sourceType, sourceID, err := parseManualLinkKey(key)
	if err != nil {
		return nil, err
	}
	row := c.db.QueryRowContext(ctx,
		`SELECT source_type, source_id, jira_key, resource_version, created_at
		FROM manual_links WHERE source_type = ? AND source_id = ?`, sourceType, sourceID)
	return scanManualLink(row)
}

func (c *manualLinkClient) List(ctx context.Context) ([]*api.ManualLink, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT source_type, source_id, jira_key, resource_version, created_at
		FROM manual_links ORDER BY source_type, source_id`)
	if err != nil {
		return nil, fmt.Errorf("list manual links: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*api.ManualLink
	for rows.Next() {
		ml, err := scanManualLinkRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ml)
	}
	return out, rows.Err()
}

func (c *manualLinkClient) Create(ctx context.Context, obj *api.ManualLink) (*api.ManualLink, error) {
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate manual link: %w", err)
	}
	now := time.Now().UTC()
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO manual_links (source_type, source_id, jira_key, resource_version, created_at)
		VALUES (?, ?, ?, 1, ?)`,
		string(obj.SourceType), obj.SourceID, obj.JiraKey, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create manual link %s/%s: %w", obj.SourceType, obj.SourceID, err)
	}
	created, err := c.Get(ctx, obj.Key())
	if err != nil {
		return nil, err
	}
	_ = c.broadcaster.Action(watch.Added, created)
	return created, nil
}

func (c *manualLinkClient) Update(ctx context.Context, obj *api.ManualLink) (*api.ManualLink, error) {
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate manual link: %w", err)
	}
	res, err := c.db.ExecContext(ctx,
		`UPDATE manual_links SET jira_key = ?, resource_version = resource_version + 1
		WHERE source_type = ? AND source_id = ? AND resource_version = ?`,
		obj.JiraKey, string(obj.SourceType), obj.SourceID, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update manual link %s/%s: %w", obj.SourceType, obj.SourceID, err)
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

func (c *manualLinkClient) Delete(ctx context.Context, key string) error {
	obj, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	sourceType, sourceID, err := parseManualLinkKey(key)
	if err != nil {
		return err
	}
	res, err := c.db.ExecContext(ctx,
		"DELETE FROM manual_links WHERE source_type = ? AND source_id = ?", sourceType, sourceID)
	if err != nil {
		return fmt.Errorf("delete manual link %s: %w", key, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	_ = c.broadcaster.Action(watch.Deleted, obj)
	return nil
}

func parseManualLinkKey(key string) (string, string, error) {
	idx := strings.Index(key, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid manual link key %q: expected sourceType/sourceID", key)
	}
	sourceType, sourceID := key[:idx], key[idx+1:]
	if len(sourceType) == 0 || len(sourceID) == 0 {
		return "", "", fmt.Errorf("invalid manual link key %q: empty source type or source ID", key)
	}
	return sourceType, sourceID, nil
}

func scanManualLink(row *sql.Row) (*api.ManualLink, error) {
	var ml api.ManualLink
	var sourceType, createdAt string
	err := row.Scan(&sourceType, &ml.SourceID, &ml.JiraKey,
		&ml.ResourceVersion, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan manual link: %w", err)
	}
	ml.SourceType = api.ManualLinkSourceType(sourceType)
	if !ml.SourceType.Valid() {
		return nil, fmt.Errorf("invalid manual link source_type %q", sourceType)
	}
	var parseErr error
	if ml.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse manual link created_at: %w", parseErr)
	}
	return &ml, nil
}

func scanManualLinkRow(rows *sql.Rows) (*api.ManualLink, error) {
	var ml api.ManualLink
	var sourceType, createdAt string
	err := rows.Scan(&sourceType, &ml.SourceID, &ml.JiraKey,
		&ml.ResourceVersion, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan manual link row: %w", err)
	}
	ml.SourceType = api.ManualLinkSourceType(sourceType)
	if !ml.SourceType.Valid() {
		return nil, fmt.Errorf("invalid manual link source_type %q", sourceType)
	}
	var parseErr error
	if ml.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse manual link created_at: %w", parseErr)
	}
	return &ml, nil
}
