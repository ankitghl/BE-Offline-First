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

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) (*domain.Item, error) {
	query := `
		INSERT INTO items (
			id, user_id, type, title, content, version, deleted, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 1, false, now(), now())
		RETURNING id, user_id, type, title, content, version, deleted, created_at, updated_at
	`
	created := &domain.Item{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		item.ID,
		item.UserID,
		item.Type,
		item.Title,
		item.Content,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.Type,
		&created.Title,
		&created.Content,
		&created.Version,
		&created.Deleted,
		&created.CreatedAt,
		&created.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *ItemRepository) ListByUser(ctx context.Context, userId string) ([]*domain.Item, error) {
	query := `
		SELECT id, user_id, type, title, content, version, deleted, created_at, updated_at
		FROM items
		WHERE user_id = $1 AND deleted = false
		ORDER BY updated_at DESC 
	`

	rows, err := r.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	items := []*domain.Item{}

	for rows.Next() {
		item := &domain.Item{}

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Type,
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

func (r *ItemRepository) GetById(ctx context.Context, userID string, id string) (*domain.Item, error) {
	query := `
		SELECT id, user_id, type, title, content, version, deleted, created_at, updated_at
		FROM items
		WHERE id = $1 AND user_id=$2
	`
	row := r.db.QueryRowContext(ctx, query, id, userID)

	var item domain.Item
	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Type,
		&item.Title,
		&item.Content,
		&item.Version,
		&item.Deleted,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &item, err
}

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) (*domain.Item, error) {
	query := `
		UPDATE items
		SET 
			title = $1,
			content = $2,
			type = $3,
			version = version + 1,
			updated_at = now()
		WHERE id = $4 
		AND user_id = $5
		AND version = $6
		AND deleted = false
	`
	res, err := r.db.ExecContext(ctx, query,
		item.Title,
		item.Content,
		item.Type,
		item.ID,
		item.UserID,
		item.Version,
	)

	if err != nil {
		return nil, err
	}

	rows, _ := res.RowsAffected()
	if rows == 1 {
		return r.GetById(ctx, item.UserID, item.ID)
	}

	// rows == 0 → explain why
	current, err := r.GetById(ctx, item.UserID, item.ID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return nil, domain.NewConflictError(current)
}

func (r *ItemRepository) SoftDelete(ctx context.Context, id string, userID string, version int) (*domain.Item, error) {
	query := `
		UPDATE items
		SET
			deleted = true,
			version = version + 1,
			updated_at = now()
		WHERE id = $1 
		AND user_id = $2
		AND version = $3		
		AND deleted = false
	`
	res, err := r.db.ExecContext(ctx, query, id, userID, version)
	if err != nil {
		return nil, err
	}

	rows, _ := res.RowsAffected()
	if rows == 1 {
		return r.GetById(ctx, userID, id)
	}

	// rows == 0 → explain
	current, err := r.GetById(ctx, userID, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return nil, domain.NewConflictError(current)
}

func (r *ItemRepository) GetChanges(ctx context.Context, userID string, sinceVersion int) ([]*domain.Item, int, error) {
	query := `
		SELECT id, user_id, type, title, content, version, deleted, created_at, updated_at
		FROM items
		WHERE user_id = $1
		And version > $2
		ORDER BY version ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, sinceVersion)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items         []*domain.Item
		latestVersion = sinceVersion
	)

	for rows.Next() {
		item := &domain.Item{}
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Type,
			&item.Title,
			&item.Content,
			&item.Version,
			&item.Deleted,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		if item.Version > latestVersion {
			latestVersion = item.Version
		}

		items = append(items, item)
	}
	return items, latestVersion, nil
}
