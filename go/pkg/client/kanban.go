package client

import (
	"context"

	"github.com/kagent-dev/kagent/go/pkg/client/api"
)

// Kanban defines the kanban operations
type Kanban interface {
	ListKanbanCards(ctx context.Context) (*api.StandardResponse[[]api.KanbanCard], error)
}

// kanbanClient handles kanban-related requests
type kanbanClient struct {
	client *BaseClient
}

// NewKanbanClient creates a new kanban client
func NewKanbanClient(client *BaseClient) Kanban {
	return &kanbanClient{client: client}
}

// ListKanbanCards lists all kanban cards
func (c *kanbanClient) ListKanbanCards(ctx context.Context) (*api.StandardResponse[[]api.KanbanCard], error) {
	resp, err := c.client.Get(ctx, "/api/kanban", "")
	if err != nil {
		return nil, err
	}

	var response api.StandardResponse[[]api.KanbanCard]
	if err := DecodeResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
