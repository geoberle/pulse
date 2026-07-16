package informer

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/api"
	"github.com/geoberle/pulse/internal/storage"
)

type Factory struct {
	store        *storage.Store
	pollInterval time.Duration
	log          logr.Logger
	mu           sync.Mutex
	started      bool
	worktrees    *Informer[*api.Worktree]
	pullRequests *Informer[*api.PullRequest]
	jiraTickets  *Informer[*api.JiraTicket]
	manualLinks  *Informer[*api.ManualLink]
}

func NewFactory(log logr.Logger, store *storage.Store, pollInterval time.Duration) *Factory {
	return &Factory{
		store:        store,
		pollInterval: pollInterval,
		log:          log,
	}
}

func (f *Factory) Worktrees() *Informer[*api.Worktree] {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.started {
		panic("informer factory: Worktrees() called after Start()")
	}
	if f.worktrees == nil {
		f.worktrees = New[*api.Worktree](
			f.log.WithName("worktrees"),
			f.store.Worktrees().List,
			f.pollInterval,
		)
	}
	return f.worktrees
}

func (f *Factory) PullRequests() *Informer[*api.PullRequest] {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.started {
		panic("informer factory: PullRequests() called after Start()")
	}
	if f.pullRequests == nil {
		f.pullRequests = New[*api.PullRequest](
			f.log.WithName("pull-requests"),
			f.store.PullRequests().List,
			f.pollInterval,
		)
	}
	return f.pullRequests
}

func (f *Factory) JiraTickets() *Informer[*api.JiraTicket] {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.started {
		panic("informer factory: JiraTickets() called after Start()")
	}
	if f.jiraTickets == nil {
		f.jiraTickets = New[*api.JiraTicket](
			f.log.WithName("jira-tickets"),
			f.store.JiraTickets().List,
			f.pollInterval,
		)
	}
	return f.jiraTickets
}

func (f *Factory) ManualLinks() *Informer[*api.ManualLink] {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.started {
		panic("informer factory: ManualLinks() called after Start()")
	}
	if f.manualLinks == nil {
		f.manualLinks = New[*api.ManualLink](
			f.log.WithName("manual-links"),
			f.store.ManualLinks().List,
			f.pollInterval,
		)
	}
	return f.manualLinks
}

func (f *Factory) Start(ctx context.Context) {
	f.mu.Lock()
	f.started = true
	informers := make([]func(context.Context), 0, 4)
	if f.worktrees != nil {
		informers = append(informers, f.worktrees.Run)
	}
	if f.pullRequests != nil {
		informers = append(informers, f.pullRequests.Run)
	}
	if f.jiraTickets != nil {
		informers = append(informers, f.jiraTickets.Run)
	}
	if f.manualLinks != nil {
		informers = append(informers, f.manualLinks.Run)
	}
	f.mu.Unlock()

	for _, run := range informers {
		go run(ctx)
	}
}

func (f *Factory) WaitForCacheSync(ctx context.Context) bool {
	f.mu.Lock()
	var checks []func() bool
	if f.worktrees != nil {
		checks = append(checks, f.worktrees.HasSynced)
	}
	if f.pullRequests != nil {
		checks = append(checks, f.pullRequests.HasSynced)
	}
	if f.jiraTickets != nil {
		checks = append(checks, f.jiraTickets.HasSynced)
	}
	if f.manualLinks != nil {
		checks = append(checks, f.manualLinks.HasSynced)
	}
	f.mu.Unlock()

	for {
		allSynced := true
		for _, check := range checks {
			if !check() {
				allSynced = false
				break
			}
		}
		if allSynced {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(10 * time.Millisecond):
		}
	}
}
