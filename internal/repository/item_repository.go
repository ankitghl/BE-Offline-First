package repository

import (
	domain "Offline-First/internal/domain/model"
	"context"
)

type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) error
	GetById(ctx context.Context, id string) (*domain.Item, error)
	ListByOwner(ctx context.Context, ownerId string) ([]*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	SoftDelete(ctx context.Context, id string) error
}
