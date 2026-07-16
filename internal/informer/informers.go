package informer

import (
	"context"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/geoberle/pulse/internal/api"
	"github.com/geoberle/pulse/internal/storage"
)

type PulseInformers interface {
	Worktrees() (cache.SharedIndexInformer, WorktreeLister)
	PullRequests() (cache.SharedIndexInformer, PullRequestLister)
	JiraTickets() (cache.SharedIndexInformer, JiraTicketLister)
	ManualLinks() (cache.SharedIndexInformer, ManualLinkLister)
	RunWithContext(ctx context.Context)
}

type pulseInformers struct {
	store        *storage.Store
	relistPeriod time.Duration
	mu           sync.Mutex
	worktrees    cache.SharedIndexInformer
	pullRequests cache.SharedIndexInformer
	jiraTickets  cache.SharedIndexInformer
	manualLinks  cache.SharedIndexInformer
	wtLister     WorktreeLister
	prLister     PullRequestLister
	jtLister     JiraTicketLister
	mlLister     ManualLinkLister
}

func NewPulseInformers(store *storage.Store, relistPeriod time.Duration) PulseInformers {
	return &pulseInformers{
		store:        store,
		relistPeriod: relistPeriod,
	}
}

func (p *pulseInformers) Worktrees() (cache.SharedIndexInformer, WorktreeLister) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.worktrees == nil {
		lw := newListWatch(
			func(ctx context.Context) (runtime.Object, error) {
				items, err := p.store.Worktrees().List(ctx)
				if err != nil {
					return nil, err
				}
				list := &api.WorktreeList{}
				for _, item := range items {
					list.Items = append(list.Items, *item)
				}
				return list, nil
			},
			p.store.WatchWorktrees,
		)
		p.worktrees = cache.NewSharedIndexInformerWithOptions(
			lw,
			&api.Worktree{},
			cache.SharedIndexInformerOptions{
				ResyncPeriod: p.relistPeriod,
				Indexers: cache.Indexers{
					ByRepo:   worktreeRepoIndexFunc,
					ByBranch: worktreeBranchIndexFunc,
				},
			},
		)
		p.wtLister = NewWorktreeLister(p.worktrees.GetIndexer())
	}
	return p.worktrees, p.wtLister
}

func (p *pulseInformers) PullRequests() (cache.SharedIndexInformer, PullRequestLister) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pullRequests == nil {
		lw := newListWatch(
			func(ctx context.Context) (runtime.Object, error) {
				items, err := p.store.PullRequests().List(ctx)
				if err != nil {
					return nil, err
				}
				list := &api.PullRequestList{}
				for _, item := range items {
					list.Items = append(list.Items, *item)
				}
				return list, nil
			},
			p.store.WatchPullRequests,
		)
		p.pullRequests = cache.NewSharedIndexInformerWithOptions(
			lw,
			&api.PullRequest{},
			cache.SharedIndexInformerOptions{
				ResyncPeriod: p.relistPeriod,
				Indexers: cache.Indexers{
					ByRepo: pullRequestRepoIndexFunc,
				},
			},
		)
		p.prLister = NewPullRequestLister(p.pullRequests.GetIndexer())
	}
	return p.pullRequests, p.prLister
}

func (p *pulseInformers) JiraTickets() (cache.SharedIndexInformer, JiraTicketLister) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.jiraTickets == nil {
		lw := newListWatch(
			func(ctx context.Context) (runtime.Object, error) {
				items, err := p.store.JiraTickets().List(ctx)
				if err != nil {
					return nil, err
				}
				list := &api.JiraTicketList{}
				for _, item := range items {
					list.Items = append(list.Items, *item)
				}
				return list, nil
			},
			p.store.WatchJiraTickets,
		)
		p.jiraTickets = cache.NewSharedIndexInformerWithOptions(
			lw,
			&api.JiraTicket{},
			cache.SharedIndexInformerOptions{
				ResyncPeriod: p.relistPeriod,
			},
		)
		p.jtLister = NewJiraTicketLister(p.jiraTickets.GetIndexer())
	}
	return p.jiraTickets, p.jtLister
}

func (p *pulseInformers) ManualLinks() (cache.SharedIndexInformer, ManualLinkLister) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.manualLinks == nil {
		lw := newListWatch(
			func(ctx context.Context) (runtime.Object, error) {
				items, err := p.store.ManualLinks().List(ctx)
				if err != nil {
					return nil, err
				}
				list := &api.ManualLinkList{}
				for _, item := range items {
					list.Items = append(list.Items, *item)
				}
				return list, nil
			},
			p.store.WatchManualLinks,
		)
		p.manualLinks = cache.NewSharedIndexInformerWithOptions(
			lw,
			&api.ManualLink{},
			cache.SharedIndexInformerOptions{
				ResyncPeriod: p.relistPeriod,
				Indexers: cache.Indexers{
					ByJiraKey:    manualLinkJiraKeyIndexFunc,
					BySourceType: manualLinkSourceTypeIndexFunc,
				},
			},
		)
		p.mlLister = NewManualLinkLister(p.manualLinks.GetIndexer())
	}
	return p.manualLinks, p.mlLister
}

func (p *pulseInformers) RunWithContext(ctx context.Context) {
	p.mu.Lock()
	var informers []cache.SharedIndexInformer
	if p.worktrees != nil {
		informers = append(informers, p.worktrees)
	}
	if p.pullRequests != nil {
		informers = append(informers, p.pullRequests)
	}
	if p.jiraTickets != nil {
		informers = append(informers, p.jiraTickets)
	}
	if p.manualLinks != nil {
		informers = append(informers, p.manualLinks)
	}
	p.mu.Unlock()

	for _, inf := range informers {
		go inf.Run(ctx.Done())
	}

	<-ctx.Done()
}

func newListWatch(listFn func(ctx context.Context) (runtime.Object, error), watchFn func() (watch.Interface, error)) cache.ListerWatcher {
	lw := &cache.ListWatch{
		ListFunc: func(_ metav1.ListOptions) (runtime.Object, error) {
			return listFn(context.Background())
		},
		WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
			return watchFn()
		},
	}
	return listWatchWithoutWatchListSemantics{lw}
}

type listWatchWithoutWatchListSemantics struct {
	*cache.ListWatch
}

func (listWatchWithoutWatchListSemantics) IsWatchListSemanticsUnSupported() bool { return true }

// Index functions

func worktreeRepoIndexFunc(obj interface{}) ([]string, error) {
	wt, ok := obj.(*api.Worktree)
	if !ok {
		return nil, nil
	}
	return []string{wt.Repo}, nil
}

func worktreeBranchIndexFunc(obj interface{}) ([]string, error) {
	wt, ok := obj.(*api.Worktree)
	if !ok {
		return nil, nil
	}
	return []string{wt.Branch}, nil
}

func pullRequestRepoIndexFunc(obj interface{}) ([]string, error) {
	pr, ok := obj.(*api.PullRequest)
	if !ok {
		return nil, nil
	}
	return []string{pr.Repo}, nil
}

func manualLinkJiraKeyIndexFunc(obj interface{}) ([]string, error) {
	ml, ok := obj.(*api.ManualLink)
	if !ok {
		return nil, nil
	}
	return []string{ml.JiraKey}, nil
}

func manualLinkSourceTypeIndexFunc(obj interface{}) ([]string, error) {
	ml, ok := obj.(*api.ManualLink)
	if !ok {
		return nil, nil
	}
	return []string{string(ml.SourceType)}, nil
}
