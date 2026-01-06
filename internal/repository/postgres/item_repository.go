package postgres

import (
	domain "Offline-First/internal/domain/model"
	"context"
	"database/sql"
	"log"
)

type ItemRepository struct {
	db *sql.DB
}

func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) NextVersion(ctx context.Context, tx *sql.Tx) (int, error) {
	var v int
	err := tx.QueryRowContext(ctx, `
		UPDATE sync_state
		SET latest_version = latest_version + 1
		WHERE id = 1
		RETURNING latest_version
	`).Scan(&v)

	return v, err
}

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) (*domain.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	version, err := r.NextVersion(ctx, tx)
	if err != nil {
		return nil, err
	}

	log.Printf("[Server] Global version allocated: %d (CREATE)", version)

	query := `
		INSERT INTO items (
			id, user_id, type, title, content, version, deleted, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, false, now(), now())
		RETURNING id, user_id, type, title, content, version, deleted, created_at, updated_at
	`
	created := &domain.Item{}

	err = tx.QueryRowContext(
		ctx,
		query,
		item.ID,
		item.UserID,
		item.Type,
		item.Title,
		item.Content,
		version,
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

	return created, tx.Commit()
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	current, err := r.GetById(ctx, item.UserID, item.ID)
	if err != nil {
		return nil, err
	}

	if current.Version != item.Version {
		return nil, domain.NewConflictError(current)
	}

	version, err := r.NextVersion(ctx, tx)
	if err != nil {
		return nil, err
	}

	log.Printf("[Server] Global version allocated: %d (UPDATE)", version)

	query := `
		UPDATE items
		SET 
			title = $1,
			content = $2,
			type = $3,
			version = $4,
			updated_at = now()
		WHERE id = $5 
		AND user_id = $6
	`
	_, err = r.db.ExecContext(ctx, query,
		item.Title,
		item.Content,
		item.Type,
		version,
		item.ID,
		item.UserID,
	)

	if err != nil {
		return nil, err
	}

	updated, err := r.GetById(ctx, item.UserID, item.ID)
	if err != nil {
		return nil, err
	}

	return updated, tx.Commit()
}

func (r *ItemRepository) SoftDelete(ctx context.Context, id string, userID string, version int) (*domain.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	current, err := r.GetById(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if current.Version != version {
		return nil, domain.NewConflictError(current)
	}

	newVersion, err := r.NextVersion(ctx, tx)
	if err != nil {
		return nil, err
	}

	log.Printf("[Server] Global version allocated: %d (DELETE)", newVersion)

	query := `
		UPDATE items
		SET
			deleted = true,
			version = $1,
			updated_at = now()
		WHERE id = $2 AND user_id = $3
	`

	_, err = tx.ExecContext(ctx, query, newVersion, id, userID)
	if err != nil {
		return nil, err
	}

	deletedItem, err := r.GetById(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	return deletedItem, tx.Commit()
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
