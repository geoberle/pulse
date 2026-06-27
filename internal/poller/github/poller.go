package github

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	gogithub "github.com/google/go-github/v72/github"

	"github.com/geoberle/pulse/internal/poller"
	"github.com/geoberle/pulse/internal/workitem"
)

// PullRequestLister is the subset of gogithub.PullRequestsService used by the poller.
type PullRequestLister interface {
	List(ctx context.Context, owner, repo string, opts *gogithub.PullRequestListOptions) ([]*gogithub.PullRequest, *gogithub.Response, error)
	ListComments(ctx context.Context, owner, repo string, number int, opts *gogithub.PullRequestListCommentsOptions) ([]*gogithub.PullRequestComment, *gogithub.Response, error)
}

// CheckRunLister is the subset of gogithub.ChecksService used by the poller.
type CheckRunLister interface {
	ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *gogithub.ListCheckRunsOptions) (*gogithub.ListCheckRunsResults, *gogithub.Response, error)
}

var _ poller.Poller = (*Poller)(nil)

type Poller struct {
	prs    PullRequestLister
	checks CheckRunLister
	repos  []string
	user   string
}

func NewPoller(prs PullRequestLister, checks CheckRunLister, repos []string, user string) *Poller {
	return &Poller{
		prs:    prs,
		checks: checks,
		repos:  repos,
		user:   user,
	}
}

func (p *Poller) Poll(ctx context.Context) ([]*workitem.WorkItem, error) {
	var items []*workitem.WorkItem
	for _, repo := range p.repos {
		owner, name, _ := strings.Cut(repo, "/")
		repoItems, err := p.pollRepo(ctx, owner, name, repo)
		if err != nil {
			return nil, fmt.Errorf("poll %s: %w", repo, err)
		}
		items = append(items, repoItems...)
	}
	return items, nil
}

func (p *Poller) pollRepo(ctx context.Context, owner, name, repo string) ([]*workitem.WorkItem, error) {
	prs, err := p.listAllPRs(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("list PRs: %w", err)
	}

	var items []*workitem.WorkItem
	for _, pr := range prs {
		if pr.User.GetLogin() != p.user {
			continue
		}
		item, err := p.buildPRItem(ctx, owner, name, repo, pr)
		if err != nil {
			return nil, fmt.Errorf("build PR #%d: %w", pr.GetNumber(), err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (p *Poller) listAllPRs(ctx context.Context, owner, repo string) ([]*gogithub.PullRequest, error) {
	var all []*gogithub.PullRequest
	opts := &gogithub.PullRequestListOptions{
		State:       "open",
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}
	for {
		prs, resp, err := p.prs.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, prs...)
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

func (p *Poller) buildPRItem(ctx context.Context, owner, name, repo string, pr *gogithub.PullRequest) (*workitem.WorkItem, error) {
	prItem, err := workitem.NewWorkItem(
		workitem.KindPR,
		fmt.Sprintf("pr:%s:%d", repo, pr.GetNumber()),
		pr.GetTitle(),
		pr.GetState(),
		&workitem.PRSpec{
			Repo:   repo,
			Number: pr.GetNumber(),
			Branch: pr.Head.GetRef(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create PR work item: %w", err)
	}

	checks, err := p.fetchChecks(ctx, owner, name, pr.Head.GetSHA())
	if err != nil {
		return nil, fmt.Errorf("fetch checks: %w", err)
	}

	reviews, err := p.fetchUnresolvedReviews(ctx, owner, name, pr.GetNumber())
	if err != nil {
		return nil, fmt.Errorf("fetch reviews: %w", err)
	}

	prItem.Children = append(prItem.Children, checks...)
	prItem.Children = append(prItem.Children, reviews...)
	return prItem, nil
}

func (p *Poller) fetchChecks(ctx context.Context, owner, repo, ref string) ([]*workitem.WorkItem, error) {
	var items []*workitem.WorkItem
	opts := &gogithub.ListCheckRunsOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}
	for {
		result, resp, err := p.checks.ListCheckRunsForRef(ctx, owner, repo, ref, opts)
		if err != nil {
			return nil, err
		}
		for _, cr := range result.CheckRuns {
			status := cr.GetConclusion()
			if len(status) == 0 {
				status = cr.GetStatus()
			}
			item, err := workitem.NewWorkItem(
				workitem.KindCheck,
				fmt.Sprintf("check:%d", cr.GetID()),
				cr.GetName(),
				status,
				&workitem.CheckSpec{Name: cr.GetName()},
			)
			if err != nil {
				return nil, fmt.Errorf("create check work item %q: %w", cr.GetName(), err)
			}
			items = append(items, item)
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return items, nil
}

func (p *Poller) fetchUnresolvedReviews(ctx context.Context, owner, repo string, number int) ([]*workitem.WorkItem, error) {
	comments, err := p.listAllComments(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}

	unresolved := findUnresolvedComments(comments, p.user)

	var items []*workitem.WorkItem
	for _, c := range unresolved {
		item, err := workitem.NewWorkItem(
			workitem.KindReview,
			fmt.Sprintf("gh-comment:%d", c.GetID()),
			c.GetPath(),
			"unresolved",
			&workitem.ReviewSpec{
				File:     c.GetPath(),
				BodyHash: bodyHash(c.GetBody()),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("create review work item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (p *Poller) listAllComments(ctx context.Context, owner, repo string, number int) ([]*gogithub.PullRequestComment, error) {
	var all []*gogithub.PullRequestComment
	opts := &gogithub.PullRequestListCommentsOptions{
		ListOptions: gogithub.ListOptions{PerPage: 100},
	}
	for {
		comments, resp, err := p.prs.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, comments...)
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// findUnresolvedComments returns thread roots that need the user's attention.
// A thread needs attention when it has non-user comments and the user hasn't
// replied after the latest one. Orphaned replies (broken InReplyTo chain)
// are treated as their own thread roots.
func findUnresolvedComments(comments []*gogithub.PullRequestComment, user string) []*gogithub.PullRequestComment {
	byID := make(map[int64]*gogithub.PullRequestComment, len(comments))
	for _, c := range comments {
		byID[c.GetID()] = c
	}

	rootOf := func(c *gogithub.PullRequestComment) int64 {
		for c.GetInReplyTo() != 0 {
			parent, ok := byID[c.GetInReplyTo()]
			if !ok {
				break
			}
			c = parent
		}
		return c.GetID()
	}

	type threadInfo struct {
		root       *gogithub.PullRequestComment
		maxUserID  int64
		maxOtherID int64
	}
	threads := make(map[int64]*threadInfo)

	for _, c := range comments {
		rootID := rootOf(c)
		ti, ok := threads[rootID]
		if !ok {
			ti = &threadInfo{}
			threads[rootID] = ti
		}
		if c.GetID() == rootID {
			ti.root = c
		}
		if c.User.GetLogin() == user {
			if c.GetID() > ti.maxUserID {
				ti.maxUserID = c.GetID()
			}
		} else {
			if c.GetID() > ti.maxOtherID {
				ti.maxOtherID = c.GetID()
			}
		}
	}

	var unresolved []*gogithub.PullRequestComment
	for _, ti := range threads {
		if ti.root == nil || ti.maxOtherID == 0 {
			continue
		}
		if ti.maxOtherID > ti.maxUserID {
			unresolved = append(unresolved, ti.root)
		}
	}
	return unresolved
}

func bodyHash(body string) string {
	h := sha256.Sum256([]byte(body))
	return hex.EncodeToString(h[:])
}
