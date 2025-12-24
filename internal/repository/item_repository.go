package repository

import (
	domain "Offline-First/internal/domain/model"
	"context"
)

type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) (*domain.Item, error)
	GetById(ctx context.Context, id string) (*domain.Item, error)
	ListByUser(ctx context.Context, userId string) ([]*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) (*domain.Item, error)
	SoftDelete(ctx context.Context, id string, userID string, version int) (*domain.Item, error)

	GetChanges(ctx context.Context, userId string, sinceVersion int) ([]*domain.Item, int, error)
}
