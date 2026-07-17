package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/geoberle/pulse/internal/api"
)

type manualLinkClient struct {
	db          *sql.DB
	broadcaster *watch.Broadcaster
}

func (c *manualLinkClient) Get(ctx context.Context, key string) (*api.ManualLink, error) {
	row := c.db.QueryRowContext(ctx,
		`SELECT name, source_type, source_id, jira_key, deletion_timestamp, finalizers,
		resource_version, created_at
		FROM manual_links WHERE name = ?`, key)
	return scanManualLink(row)
}

func (c *manualLinkClient) List(ctx context.Context) ([]*api.ManualLink, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT name, source_type, source_id, jira_key, deletion_timestamp, finalizers,
		resource_version, created_at
		FROM manual_links ORDER BY name`)
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
	obj.Name = api.ManualLinkName(obj.SourceType, obj.SourceID)
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate manual link: %w", err)
	}
	now := time.Now().UTC()
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO manual_links (name, source_type, source_id, jira_key, finalizers,
		resource_version, created_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)`,
		obj.Name, string(obj.SourceType), obj.SourceID, obj.JiraKey,
		finalizersJSON, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create manual link %s: %w", obj.Name, err)
	}
	created, err := c.Get(ctx, obj.Name)
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
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	var deletionTS *string
	if obj.DeletionTimestamp != nil {
		s := obj.DeletionTimestamp.UTC().Format(time.RFC3339Nano)
		deletionTS = &s
	}
	res, err := c.db.ExecContext(ctx,
		`UPDATE manual_links SET source_type = ?, source_id = ?, jira_key = ?,
		deletion_timestamp = ?, finalizers = ?,
		resource_version = resource_version + 1
		WHERE name = ? AND resource_version = ?`,
		string(obj.SourceType), obj.SourceID, obj.JiraKey,
		deletionTS, finalizersJSON,
		obj.Name, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update manual link %s: %w", obj.Name, err)
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
		if _, err := c.db.ExecContext(ctx, "DELETE FROM manual_links WHERE name = ?", obj.Name); err != nil {
			return nil, fmt.Errorf("garbage collect manual link %s: %w", obj.Name, err)
		}
		_ = c.broadcaster.Action(watch.Deleted, updated)
		return updated, nil
	}
	_ = c.broadcaster.Action(watch.Modified, updated)
	return updated, nil
}

func (c *manualLinkClient) Delete(ctx context.Context, key string) error {
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
				`UPDATE manual_links SET deletion_timestamp = ?, resource_version = resource_version + 1
				WHERE name = ? AND resource_version = ?`,
				now.Format(time.RFC3339Nano), key, obj.ResourceVersion)
			if err != nil {
				return fmt.Errorf("soft delete manual link %s: %w", key, err)
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
		res, err := c.db.ExecContext(ctx, "DELETE FROM manual_links WHERE name = ?", key)
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
	return ErrConflict
}

func scanManualLink(row *sql.Row) (*api.ManualLink, error) {
	var ml api.ManualLink
	var sourceType, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := row.Scan(&ml.Name, &sourceType, &ml.SourceID, &ml.JiraKey,
		&deletionTS, &finalizersJSON, &ml.ResourceVersion, &createdAt)
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
	if err := parseScanFields(&ml.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("manual link %s: %w", ml.Name, err)
	}
	return &ml, nil
}

func scanManualLinkRow(rows *sql.Rows) (*api.ManualLink, error) {
	var ml api.ManualLink
	var sourceType, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := rows.Scan(&ml.Name, &sourceType, &ml.SourceID, &ml.JiraKey,
		&deletionTS, &finalizersJSON, &ml.ResourceVersion, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan manual link row: %w", err)
	}
	ml.SourceType = api.ManualLinkSourceType(sourceType)
	if !ml.SourceType.Valid() {
		return nil, fmt.Errorf("invalid manual link source_type %q", sourceType)
	}
	if err := parseScanFields(&ml.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("manual link %s: %w", ml.Name, err)
	}
	return &ml, nil
}
