package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/geoberle/pulse/internal/api"

	_ "modernc.org/sqlite"
)

type Store struct {
	db           *sql.DB
	worktrees    *worktreeClient
	pullRequests *pullRequestClient
	jiraTickets  *jiraTicketClient
	manualLinks  *manualLinkClient
}

func New(ctx context.Context, dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("create state directory %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", dbPath, err)
	}

	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	if dbPath != ":memory:" {
		var journalMode string
		if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set WAL mode: %w", err)
		}
		if journalMode != "wal" {
			_ = db.Close()
			return nil, fmt.Errorf("WAL mode not supported: got %q", journalMode)
		}
	}

	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	s := &Store{db: db}
	s.worktrees = &worktreeClient{db: db}
	s.pullRequests = &pullRequestClient{db: db}
	s.jiraTickets = &jiraTicketClient{db: db}
	s.manualLinks = &manualLinkClient{db: db}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Worktrees() Table[*api.Worktree] {
	return s.worktrees
}

func (s *Store) PullRequests() Table[*api.PullRequest] {
	return s.pullRequests
}

func (s *Store) JiraTickets() Table[*api.JiraTicket] {
	return s.jiraTickets
}

func (s *Store) ManualLinks() Table[*api.ManualLink] {
	return s.manualLinks
}
