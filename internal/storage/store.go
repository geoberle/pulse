package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/geoberle/pulse/internal/api"

	_ "modernc.org/sqlite"
)

type Store struct {
	db           *sql.DB
	worktrees    *worktreeClient
	pullRequests *pullRequestClient
	jiraTickets  *jiraTicketClient
	manualLinks  *manualLinkClient

	wtBroadcaster *watch.Broadcaster
	prBroadcaster *watch.Broadcaster
	jtBroadcaster *watch.Broadcaster
	mlBroadcaster *watch.Broadcaster
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

	s := &Store{
		db:            db,
		wtBroadcaster: watch.NewBroadcaster(100, watch.WaitIfChannelFull),
		prBroadcaster: watch.NewBroadcaster(100, watch.WaitIfChannelFull),
		jtBroadcaster: watch.NewBroadcaster(100, watch.WaitIfChannelFull),
		mlBroadcaster: watch.NewBroadcaster(100, watch.WaitIfChannelFull),
	}
	s.worktrees = &worktreeClient{db: db, broadcaster: s.wtBroadcaster}
	s.pullRequests = &pullRequestClient{db: db, broadcaster: s.prBroadcaster}
	s.jiraTickets = &jiraTicketClient{db: db, broadcaster: s.jtBroadcaster}
	s.manualLinks = &manualLinkClient{db: db, broadcaster: s.mlBroadcaster}
	return s, nil
}

func (s *Store) Close() error {
	s.wtBroadcaster.Shutdown()
	s.prBroadcaster.Shutdown()
	s.jtBroadcaster.Shutdown()
	s.mlBroadcaster.Shutdown()
	return s.db.Close()
}

func (s *Store) WatchWorktrees() (watch.Interface, error) {
	return s.wtBroadcaster.Watch()
}

func (s *Store) WatchPullRequests() (watch.Interface, error) {
	return s.prBroadcaster.Watch()
}

func (s *Store) WatchJiraTickets() (watch.Interface, error) {
	return s.jtBroadcaster.Watch()
}

func (s *Store) WatchManualLinks() (watch.Interface, error) {
	return s.mlBroadcaster.Watch()
}

func (s *Store) Worktrees() Client[*api.Worktree] {
	return s.worktrees
}

func (s *Store) PullRequests() Client[*api.PullRequest] {
	return s.pullRequests
}

func (s *Store) JiraTickets() Client[*api.JiraTicket] {
	return s.jiraTickets
}

func (s *Store) ManualLinks() Client[*api.ManualLink] {
	return s.manualLinks
}
