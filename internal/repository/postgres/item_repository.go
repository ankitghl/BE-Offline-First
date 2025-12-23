package postgres

import (
	domain "Offline-First/internal/domain/model"
	"context"
	"database/sql"
)

type ItemRepository struct {
	db *sql.DB
}

func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) error {
	query := `
		INSERT INTO items (
			id, owner_id, title, content, version, deleted, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`

	_, err := r.db.ExecContext(ctx, query, item.ID, item.OwnerId, item.Title, item.Content, item.Version, item.Deleted)

	return err
}

func (r *ItemRepository) GetById(ctx context.Context, id string) (*domain.Item, error) {
	query := `
		SELECT id, owner_id, title, content, version, deleted, created_at, updated_at
		FROM items
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var item domain.Item
	err := row.Scan(
		&item.ID,
		&item.OwnerId,
		&item.Title,
		&item.Content,
		&item.Version,
		&item.Deleted,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &item, err
}

func (r *ItemRepository) ListByOwner(ctx context.Context, ownerId string) ([]*domain.Item, error) {
	query := `
		SELECT id, owner_id, title, content, version, deleted, created_at, updated_at
		FROM items
		WHERE owner_id = $1 AND deleted = false
		ORDER BY updated_at DESC 
	`

	rows, err := r.db.QueryContext(ctx, query, ownerId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	items := []*domain.Item{}

	for rows.Next() {
		item := &domain.Item{}

		err := rows.Scan(
			&item.ID,
			&item.OwnerId,
			&item.Title,
			&item.Content,
			&item.Version,
			&item.Deleted,
			&item.CreatedAt,
			&item.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) error {
	query := `
		UPDATE items
		SET 
			title = $1,
			content = $2,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at = false
	`

	result, err := r.db.ExecContext(ctx, query, item.Title, item.Content, item.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ItemRepository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE items
		SET
			deleted = true,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND deleted = false
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
