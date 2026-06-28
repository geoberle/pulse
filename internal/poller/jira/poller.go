package jira

import (
	"context"
	"fmt"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"

	"github.com/geoberle/pulse/internal/poller"
	"github.com/geoberle/pulse/internal/workitem"
)

// IssueSearcher is the subset of the go-atlassian search service used by the poller.
type IssueSearcher interface {
	SearchJQL(ctx context.Context, jql string, fields, expands []string, maxResults int, nextPageToken string) (*models.IssueSearchJQLSchemeV2, *models.ResponseScheme, error)
}

var _ poller.Poller = (*Poller)(nil)

type Poller struct {
	search         IssueSearcher
	project        string
	staleThreshold time.Duration
	now            func() time.Time
}

func NewPoller(search IssueSearcher, project string, staleThreshold time.Duration) *Poller {
	return &Poller{
		search:         search,
		project:        project,
		staleThreshold: staleThreshold,
		now:            time.Now,
	}
}

func (p *Poller) Poll(ctx context.Context) ([]*workitem.WorkItem, error) {
	jql := fmt.Sprintf(
		`project = %s AND (assignee = currentUser() OR creator = currentUser()) AND type in (Story, Bug) AND status != Closed`,
		p.project,
	)
	fields := []string{"summary", "status", "issuetype", "updated"}

	var items []*workitem.WorkItem
	nextPage := ""
	for {
		result, resp, err := p.search.SearchJQL(ctx, jql, fields, nil, 50, nextPage)
		if err != nil {
			statusCode := 0
			if resp != nil {
				statusCode = resp.Code
			}
			return nil, fmt.Errorf("jira search failed (HTTP %d): %w", statusCode, err)
		}

		for _, issue := range result.Issues {
			item, err := p.buildItem(issue)
			if err != nil {
				return nil, fmt.Errorf("build item %s: %w", issue.Key, err)
			}
			items = append(items, item)
		}

		if len(result.NextPageToken) == 0 {
			break
		}
		nextPage = result.NextPageToken
	}
	return items, nil
}

func (p *Poller) buildItem(issue *models.IssueSchemeV2) (*workitem.WorkItem, error) {
	var staleness workitem.StalenessState
	var status, summary string

	if f := issue.Fields; f != nil {
		summary = f.Summary
		if f.Status != nil {
			status = f.Status.Name
		}
		if f.Updated != nil {
			updated := time.Time(*f.Updated)
			if p.now().Sub(updated) > p.staleThreshold {
				staleness = workitem.StalenessStale
			} else {
				staleness = workitem.StalenessActive
			}
		}
	}

	return workitem.NewWorkItem(
		workitem.KindJira,
		fmt.Sprintf("jira:%s", issue.Key),
		summary,
		status,
		&workitem.JiraSpec{
			Key:       issue.Key,
			Staleness: staleness,
		},
	)
}
