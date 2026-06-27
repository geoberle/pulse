package poller

import (
	"context"

	"github.com/geoberle/pulse/internal/workitem"
)

// Poller fetches work items from an external source. Implementations return
// a flat list of top-level items (e.g. PRs with Check/Review children).
// The engine assembles these into the full tree (grouping under Jira items).
type Poller interface {
	Poll(ctx context.Context) ([]*workitem.WorkItem, error)
}
