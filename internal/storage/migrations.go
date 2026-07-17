package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

var migrations = []string{
	// migration 001: initial schema
	`CREATE TABLE worktrees (
		name TEXT PRIMARY KEY,
		repo TEXT NOT NULL,
		branch TEXT NOT NULL,
		commit_state TEXT NOT NULL DEFAULT '',
		deletion_timestamp TEXT,
		finalizers TEXT NOT NULL DEFAULT '[]',
		resource_version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL
	);

	CREATE TABLE pull_requests (
		name TEXT PRIMARY KEY,
		repo TEXT NOT NULL,
		number INTEGER NOT NULL,
		branch TEXT NOT NULL,
		url TEXT NOT NULL,
		status TEXT NOT NULL,
		ci_status TEXT NOT NULL,
		review_status TEXT NOT NULL,
		unresolved_comments INTEGER NOT NULL DEFAULT 0,
		author TEXT NOT NULL,
		deletion_timestamp TEXT,
		finalizers TEXT NOT NULL DEFAULT '[]',
		resource_version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL
	);

	CREATE TABLE jira_tickets (
		name TEXT PRIMARY KEY,
		summary TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		issue_type TEXT NOT NULL,
		epic_key TEXT NOT NULL DEFAULT '',
		last_activity TEXT NOT NULL,
		deletion_timestamp TEXT,
		finalizers TEXT NOT NULL DEFAULT '[]',
		resource_version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL
	);

	CREATE TABLE manual_links (
		name TEXT PRIMARY KEY,
		source_type TEXT NOT NULL,
		source_id TEXT NOT NULL,
		jira_key TEXT NOT NULL,
		deletion_timestamp TEXT,
		finalizers TEXT NOT NULL DEFAULT '[]',
		resource_version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL
	);

	CREATE INDEX idx_manual_links_jira_key ON manual_links(jira_key);`,
}

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	var current int
	row := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM migrations")
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}

	if current > len(migrations) {
		return fmt.Errorf("migration version %d exceeds known migrations %d; possible corruption", current, len(migrations))
	}

	for i := current; i < len(migrations); i++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("migration %d canceled: %w", i+1, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec("INSERT INTO migrations (version, applied_at) VALUES (?, ?)",
			i+1, time.Now().UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", i+1, err)
		}
	}
	return nil
}
