package storage

import (
	"context"

	"github.com/geoberle/pulse/internal/api"
)

type Client[T any] interface {
	Get(ctx context.Context, key string) (T, error)
	List(ctx context.Context) ([]T, error)
	Create(ctx context.Context, obj T) (T, error)
	Update(ctx context.Context, obj T) (T, error)
	Delete(ctx context.Context, key string) error
}

type StoreClient interface {
	Worktrees() Client[*api.Worktree]
	PullRequests() Client[*api.PullRequest]
	JiraTickets() Client[*api.JiraTicket]
	ManualLinks() Client[*api.ManualLink]
}
