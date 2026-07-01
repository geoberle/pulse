package jira

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	var status workitem.WorkItemPhase
	var summary string

	if f := issue.Fields; f != nil {
		summary = f.Summary
		if f.Status != nil {
			status = workitem.WorkItemPhase(f.Status.Name)
		}
	}

	name := fmt.Sprintf("jira.%s", strings.ToLower(issue.Key))
	item, err := workitem.NewWorkItem(
		workitem.KindJira,
		name,
		status,
		&workitem.JiraSpec{
			Key:     issue.Key,
			Summary: summary,
		},
	)
	if err != nil {
		return nil, err
	}

	if f := issue.Fields; f != nil && f.Updated != nil {
		updated := time.Time(*f.Updated)
		stale := p.now().Sub(updated) > p.staleThreshold
		condStatus := metav1.ConditionFalse
		reason := string(workitem.ReasonWithinThreshold)
		if stale {
			condStatus = metav1.ConditionTrue
			reason = string(workitem.ReasonThresholdExceeded)
		}
		meta.SetStatusCondition(&item.Status.Conditions, metav1.Condition{
			Type:               string(workitem.ConditionStale),
			Status:             condStatus,
			Reason:             reason,
			LastTransitionTime: metav1.Now(),
		})
	}

	return item, nil
}
