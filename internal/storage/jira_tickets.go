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
		`SELECT key, summary, description, status, issue_type, epic_key,
		last_activity, last_seen, resource_version, created_at
		FROM jira_tickets WHERE key = ?`, key)
	return scanJiraTicket(row)
}

func (c *jiraTicketClient) List(ctx context.Context) ([]*api.JiraTicket, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT key, summary, description, status, issue_type, epic_key,
		last_activity, last_seen, resource_version, created_at
		FROM jira_tickets ORDER BY key`)
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
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO jira_tickets (key, summary, description, status, issue_type,
		epic_key, last_activity, last_seen, resource_version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)`,
		obj.TicketKey, obj.Summary, obj.Description, string(obj.Status), string(obj.IssueType),
		obj.EpicKey, obj.LastActivity.UTC().Format(time.RFC3339Nano),
		obj.LastSeen.UTC().Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create jira ticket %s: %w", obj.TicketKey, err)
	}
	created, err := c.Get(ctx, obj.TicketKey)
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
	res, err := c.db.ExecContext(ctx,
		`UPDATE jira_tickets SET summary = ?, description = ?, status = ?,
		issue_type = ?, epic_key = ?, last_activity = ?, last_seen = ?,
		resource_version = resource_version + 1
		WHERE key = ? AND resource_version = ?`,
		obj.Summary, obj.Description, string(obj.Status), string(obj.IssueType), obj.EpicKey,
		obj.LastActivity.UTC().Format(time.RFC3339Nano),
		obj.LastSeen.UTC().Format(time.RFC3339Nano),
		obj.TicketKey, obj.ResourceVersion)
	if err != nil {
		return nil, fmt.Errorf("update jira ticket %s: %w", obj.TicketKey, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := c.Get(ctx, obj.TicketKey); err != nil {
			return nil, ErrNotFound
		}
		return nil, ErrConflict
	}
	updated, err := c.Get(ctx, obj.TicketKey)
	if err != nil {
		return nil, err
	}
	_ = c.broadcaster.Action(watch.Modified, updated)
	return updated, nil
}

func (c *jiraTicketClient) Delete(ctx context.Context, key string) error {
	obj, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	res, err := c.db.ExecContext(ctx, "DELETE FROM jira_tickets WHERE key = ?", key)
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

func scanJiraTicket(row *sql.Row) (*api.JiraTicket, error) {
	var jt api.JiraTicket
	var status, issueType, lastActivity, lastSeen, createdAt string
	err := row.Scan(&jt.TicketKey, &jt.Summary, &jt.Description, &status,
		&issueType, &jt.EpicKey, &lastActivity, &lastSeen,
		&jt.ResourceVersion, &createdAt)
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
	if jt.LastSeen, parseErr = time.Parse(time.RFC3339Nano, lastSeen); parseErr != nil {
		return nil, fmt.Errorf("parse jira ticket last_seen: %w", parseErr)
	}
	if jt.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse jira ticket created_at: %w", parseErr)
	}
	return &jt, nil
}

func scanJiraTicketRow(rows *sql.Rows) (*api.JiraTicket, error) {
	var jt api.JiraTicket
	var status, issueType, lastActivity, lastSeen, createdAt string
	err := rows.Scan(&jt.TicketKey, &jt.Summary, &jt.Description, &status,
		&issueType, &jt.EpicKey, &lastActivity, &lastSeen,
		&jt.ResourceVersion, &createdAt)
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
	if jt.LastSeen, parseErr = time.Parse(time.RFC3339Nano, lastSeen); parseErr != nil {
		return nil, fmt.Errorf("parse jira ticket last_seen: %w", parseErr)
	}
	if jt.CreationTimestamp, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
		return nil, fmt.Errorf("parse jira ticket created_at: %w", parseErr)
	}
	return &jt, nil
}
