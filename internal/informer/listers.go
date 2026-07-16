package informer

import (
	"fmt"

	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/api"
)

const (
	ByRepo       = "byRepo"
	ByBranch     = "byBranch"
	ByJiraKey    = "byJiraKey"
	BySourceType = "bySourceType"
)

func listAll[T any](store cache.Store) ([]*T, error) {
	items := store.List()
	result := make([]*T, 0, len(items))
	for _, item := range items {
		typed, ok := item.(*T)
		if !ok {
			return nil, fmt.Errorf("expected *%T, got %T", *new(T), item)
		}
		result = append(result, typed)
	}
	return result, nil
}

func getByKey[T any](indexer cache.Indexer, key string) (*T, bool, error) {
	item, exists, err := indexer.GetByKey(key)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	typed, ok := item.(*T)
	if !ok {
		return nil, false, fmt.Errorf("expected *%T, got %T", *new(T), item)
	}
	return typed, true, nil
}

func listFromIndex[T any](indexer cache.Indexer, indexName, key string) ([]*T, error) {
	items, err := indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result := make([]*T, 0, len(items))
	for _, item := range items {
		typed, ok := item.(*T)
		if !ok {
			return nil, fmt.Errorf("expected *%T, got %T", *new(T), item)
		}
		result = append(result, typed)
	}
	return result, nil
}

type WorktreeLister interface {
	List() ([]*api.Worktree, error)
	Get(key string) (*api.Worktree, bool, error)
	ByIndex(indexName, key string) ([]*api.Worktree, error)
}

type worktreeLister struct {
	indexer cache.Indexer
}

func NewWorktreeLister(indexer cache.Indexer) WorktreeLister {
	return &worktreeLister{indexer: indexer}
}

func (l *worktreeLister) List() ([]*api.Worktree, error) {
	return listAll[api.Worktree](l.indexer)
}

func (l *worktreeLister) Get(key string) (*api.Worktree, bool, error) {
	return getByKey[api.Worktree](l.indexer, key)
}

func (l *worktreeLister) ByIndex(indexName, key string) ([]*api.Worktree, error) {
	return listFromIndex[api.Worktree](l.indexer, indexName, key)
}

type PullRequestLister interface {
	List() ([]*api.PullRequest, error)
	Get(key string) (*api.PullRequest, bool, error)
	ByIndex(indexName, key string) ([]*api.PullRequest, error)
}

type pullRequestLister struct {
	indexer cache.Indexer
}

func NewPullRequestLister(indexer cache.Indexer) PullRequestLister {
	return &pullRequestLister{indexer: indexer}
}

func (l *pullRequestLister) List() ([]*api.PullRequest, error) {
	return listAll[api.PullRequest](l.indexer)
}

func (l *pullRequestLister) Get(key string) (*api.PullRequest, bool, error) {
	return getByKey[api.PullRequest](l.indexer, key)
}

func (l *pullRequestLister) ByIndex(indexName, key string) ([]*api.PullRequest, error) {
	return listFromIndex[api.PullRequest](l.indexer, indexName, key)
}

type JiraTicketLister interface {
	List() ([]*api.JiraTicket, error)
	Get(key string) (*api.JiraTicket, bool, error)
	ByIndex(indexName, key string) ([]*api.JiraTicket, error)
}

type jiraTicketLister struct {
	indexer cache.Indexer
}

func NewJiraTicketLister(indexer cache.Indexer) JiraTicketLister {
	return &jiraTicketLister{indexer: indexer}
}

func (l *jiraTicketLister) List() ([]*api.JiraTicket, error) {
	return listAll[api.JiraTicket](l.indexer)
}

func (l *jiraTicketLister) Get(key string) (*api.JiraTicket, bool, error) {
	return getByKey[api.JiraTicket](l.indexer, key)
}

func (l *jiraTicketLister) ByIndex(indexName, key string) ([]*api.JiraTicket, error) {
	return listFromIndex[api.JiraTicket](l.indexer, indexName, key)
}

type ManualLinkLister interface {
	List() ([]*api.ManualLink, error)
	Get(key string) (*api.ManualLink, bool, error)
	ByIndex(indexName, key string) ([]*api.ManualLink, error)
}

type manualLinkLister struct {
	indexer cache.Indexer
}

func NewManualLinkLister(indexer cache.Indexer) ManualLinkLister {
	return &manualLinkLister{indexer: indexer}
}

func (l *manualLinkLister) List() ([]*api.ManualLink, error) {
	return listAll[api.ManualLink](l.indexer)
}

func (l *manualLinkLister) Get(key string) (*api.ManualLink, bool, error) {
	return getByKey[api.ManualLink](l.indexer, key)
}

func (l *manualLinkLister) ByIndex(indexName, key string) ([]*api.ManualLink, error) {
	return listFromIndex[api.ManualLink](l.indexer, indexName, key)
}
