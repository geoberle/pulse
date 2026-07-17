package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/geoberle/pulse/internal/api"
)

type jiraTicketClient struct {
	db          *sql.DB
	broadcaster *watch.Broadcaster
}

func (c *jiraTicketClient) Get(ctx context.Context, key string) (*api.JiraTicket, error) {
	row := c.db.QueryRowContext(ctx,
		`SELECT name, summary, description, status, issue_type, epic_key,
		last_activity, deletion_timestamp, finalizers, resource_version, created_at
		FROM jira_tickets WHERE name = ?`, key)
	return scanJiraTicket(row)
}

func (c *jiraTicketClient) List(ctx context.Context) ([]*api.JiraTicket, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT name, summary, description, status, issue_type, epic_key,
		last_activity, deletion_timestamp, finalizers, resource_version, created_at
		FROM jira_tickets ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list jira tickets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*api.JiraTicket
	for rows.Next() {
		jt, err := scanJiraTicketRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, jt)
	}
	return out, rows.Err()
}

func (c *jiraTicketClient) Create(ctx context.Context, obj *api.JiraTicket) (*api.JiraTicket, error) {
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate jira ticket: %w", err)
	}
	now := time.Now().UTC()
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO jira_tickets (name, summary, description, status, issue_type,
		epic_key, last_activity, finalizers, resource_version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)`,
		obj.Name, obj.Summary, obj.Description, string(obj.Status), string(obj.IssueType),
		obj.EpicKey, obj.LastActivity.UTC().Format(time.RFC3339Nano),
		finalizersJSON, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create jira ticket %s: %w", obj.Name, err)
	}
	created, err := c.Get(ctx, obj.Name)
	if err != nil {
		return nil, err
	}
	_ = c.broadcaster.Action(watch.Added, created)
	return created, nil
}

func (c *jiraTicketClient) Update(ctx context.Context, obj *api.JiraTicket) (*api.JiraTicket, error) {
	if err := obj.Validate(); err != nil {
		return nil, fmt.Errorf("validate jira ticket: %w", err)
	}
	finalizersJSON := marshalFinalizers(obj.Finalizers)
	var deletionTS *string
	if obj.DeletionTimestamp != nil {
		s := obj.DeletionTimestamp.UTC().Format(time.RFC3339Nano)
		deletionTS = &s
	}
	res, err := c.db.ExecContext(ctx,
		`UPDATE jira_tickets SET summary = ?, description = ?, status = ?,
		issue_type = ?, epic_key = ?, last_activity = ?,
		deletion_timestamp = ?, finalizers = ?,
		resource_version = resource_version + 1
		WHERE name = ? AND resource_version = ?`,
		obj.Summary, obj.Description, string(obj.Status), string(obj.IssueType), obj.EpicKey,
		obj.LastActivity.UTC().Format(time.RFC3339Nano),
		deletionTS, finalizersJSON,
		obj.Name, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update jira ticket %s: %w", obj.Name, err)
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
		if _, err := c.db.ExecContext(ctx, "DELETE FROM jira_tickets WHERE name = ?", obj.Name); err != nil {
			return nil, fmt.Errorf("garbage collect jira ticket %s: %w", obj.Name, err)
		}
		_ = c.broadcaster.Action(watch.Deleted, updated)
		return updated, nil
	}
	_ = c.broadcaster.Action(watch.Modified, updated)
	return updated, nil
}

func (c *jiraTicketClient) Delete(ctx context.Context, key string) error {
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
				`UPDATE jira_tickets SET deletion_timestamp = ?, resource_version = resource_version + 1
				WHERE name = ? AND resource_version = ?`,
				now.Format(time.RFC3339Nano), key, obj.ResourceVersion)
			if err != nil {
				return fmt.Errorf("soft delete jira ticket %s: %w", key, err)
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
		res, err := c.db.ExecContext(ctx, "DELETE FROM jira_tickets WHERE name = ?", key)
		if err != nil {
			return fmt.Errorf("delete jira ticket %s: %w", key, err)
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

func scanJiraTicket(row *sql.Row) (*api.JiraTicket, error) {
	var jt api.JiraTicket
	var status, issueType, lastActivity, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := row.Scan(&jt.Name, &jt.Summary, &jt.Description, &status,
		&issueType, &jt.EpicKey, &lastActivity,
		&deletionTS, &finalizersJSON, &jt.ResourceVersion, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan jira ticket: %w", err)
	}
	jt.Status = api.JiraStatus(status)
	if !jt.Status.Valid() {
		return nil, fmt.Errorf("invalid jira ticket status %q", status)
	}
	jt.IssueType = api.JiraIssueType(issueType)
	if !jt.IssueType.Valid() {
		return nil, fmt.Errorf("invalid jira ticket issue_type %q", issueType)
	}
	var parseErr error
	if jt.LastActivity, parseErr = time.Parse(time.RFC3339Nano, lastActivity); parseErr != nil {
		return nil, fmt.Errorf("parse jira ticket last_activity: %w", parseErr)
	}
	if err := parseScanFields(&jt.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("jira ticket %s: %w", jt.Name, err)
	}
	return &jt, nil
}

func scanJiraTicketRow(rows *sql.Rows) (*api.JiraTicket, error) {
	var jt api.JiraTicket
	var status, issueType, lastActivity, finalizersJSON, createdAt string
	var deletionTS sql.NullString
	err := rows.Scan(&jt.Name, &jt.Summary, &jt.Description, &status,
		&issueType, &jt.EpicKey, &lastActivity,
		&deletionTS, &finalizersJSON, &jt.ResourceVersion, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("scan jira ticket row: %w", err)
	}
	jt.Status = api.JiraStatus(status)
	if !jt.Status.Valid() {
		return nil, fmt.Errorf("invalid jira ticket status %q", status)
	}
	jt.IssueType = api.JiraIssueType(issueType)
	if !jt.IssueType.Valid() {
		return nil, fmt.Errorf("invalid jira ticket issue_type %q", issueType)
	}
	var parseErr error
	if jt.LastActivity, parseErr = time.Parse(time.RFC3339Nano, lastActivity); parseErr != nil {
		return nil, fmt.Errorf("parse jira ticket last_activity: %w", parseErr)
	}
	if err := parseScanFields(&jt.ObjectMeta, deletionTS, finalizersJSON, createdAt); err != nil {
		return nil, fmt.Errorf("jira ticket %s: %w", jt.Name, err)
	}
	return &jt, nil
}
