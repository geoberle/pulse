package storage

import (
	"context"

	"github.com/geoberle/pulse/internal/api"
)

type Table[T api.Object] interface {
	Get(ctx context.Context, key string) (T, error)
	List(ctx context.Context) ([]T, error)
	Create(ctx context.Context, obj T) (T, error)
	Update(ctx context.Context, obj T) (T, error)
	Delete(ctx context.Context, key string) error
}

type StoreClient interface {
	Worktrees() Table[*api.Worktree]
	PullRequests() Table[*api.PullRequest]
	JiraTickets() Table[*api.JiraTicket]
	ManualLinks() Table[*api.ManualLink]
}
