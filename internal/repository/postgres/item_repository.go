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

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item, mutationID string) (*domain.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1️⃣ Idempotency check
	if appliedVersion, ok, err := r.getAppliedVersion(ctx, tx, mutationID); err != nil {
		return nil, err
	} else if ok {
		// Mutation already applied → return existing item
		existing, err := r.GetByIdTx(ctx, tx, item.UserID, item.ID)
		if err != nil {
			return nil, err
		}
		existing.Version = appliedVersion
		return existing, nil
	}

	// 2️⃣ Ensure item does not already exist (defensive)
	_, err = r.GetByIdTx(ctx, tx, item.UserID, item.ID)
	if err == nil {
		return nil, domain.ErrAlreadyExists
	}
	if err != domain.ErrNotFound {
		return nil, err
	}

	// 3️⃣ Allocate global version
	version, err := r.NextVersion(ctx, tx)
	if err != nil {
		return nil, err
	}

	log.Printf("[Server] Global version allocated: %d (CREATE)", version)

	query := `
		INSERT INTO items (
			id, user_id, type, title, content, version, deleted, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, false, now(), now())
	`
	_, err = tx.ExecContext(
		ctx,
		query,
		item.ID,
		item.UserID,
		item.Type,
		item.Title,
		item.Content,
		version,
	)

	if err != nil {
		return nil, err
	}

	// 5️⃣ Record mutation
	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO mutation_log (
			mutation_id,
			item_id,
			mutation_type,
			applied_version
		)
		VALUES ($1, $2, 'create', $3)
		`,
		mutationID,
		item.ID,
		version,
	)
	if err != nil {
		return nil, err
	}

	created, err := r.GetByIdTx(ctx, tx, item.UserID, item.ID)
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

func (r *ItemRepository) GetByIdTx(ctx context.Context, tx *sql.Tx,
	userID string, id string) (*domain.Item, error) {
	query := `
		SELECT id, user_id, type, title, content, version, deleted, created_at, updated_at
		FROM items
		WHERE id = $1 AND user_id=$2
	`
	row := tx.QueryRowContext(ctx, query, id, userID)

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

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item, mutationID string) (*domain.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1️⃣ Idempotency check
	if appliedVersion, ok, err := r.getAppliedVersion(ctx, tx, mutationID); err != nil {
		return nil, err
	} else if ok {
		// Already applied → return current item state
		current, err := r.GetByIdTx(ctx, tx, item.UserID, item.ID)
		if err != nil {
			return nil, err
		}
		current.Version = appliedVersion
		return current, nil
	}

	// 2️⃣ Load current state
	current, err := r.GetByIdTx(ctx, tx, item.UserID, item.ID)
	if err != nil {
		return nil, err
	}

	if current.Version != item.Version {
		return nil, domain.NewConflictError(current)
	}

	// 3️⃣ Allocate global version
	newVersion, err := r.NextVersion(ctx, tx)
	if err != nil {
		return nil, err
	}

	log.Printf("[Server] Global version allocated: %d (UPDATE)", newVersion)

	query := `
		UPDATE items
		SET 
			title = $1,
			content = $2,
			type = $3,
			version = $4,
			deleted = false,
			updated_at = now()
		WHERE id = $5 
		AND user_id = $6
	`
	_, err = r.db.ExecContext(ctx, query,
		item.Title,
		item.Content,
		item.Type,
		newVersion,
		item.ID,
		item.UserID,
	)

	if err != nil {
		return nil, err
	}

	// 5️⃣ Record mutation
	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO mutation_log (
			mutation_id,
			item_id,
			mutation_type,
			applied_version
		)
		VALUES ($1, $2, 'update', $3)
		`,
		mutationID,
		item.ID,
		newVersion,
	)
	if err != nil {
		return nil, err
	}

	updated, err := r.GetByIdTx(ctx, tx, item.UserID, item.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *ItemRepository) SoftDelete(ctx context.Context, id string, userID string, version int, mutationID string) (*domain.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1️⃣ Idempotency check
	if appliedVersion, ok, err := r.getAppliedVersion(ctx, tx, mutationID); err != nil {
		return nil, err
	} else if ok {
		current, err := r.GetByIdTx(ctx, tx, userID, id)
		if err != nil {
			return nil, err
		}
		current.Version = appliedVersion
		return current, nil
	}

	// 2️⃣ Load current state
	current, err := r.GetByIdTx(ctx, tx, userID, id)
	if err != nil {
		return nil, err
	}

	if current.Version != version {
		return nil, domain.NewConflictError(current)
	}

	// 3️⃣ Allocate global version
	newVersion, err := r.NextVersion(ctx, tx)
	if err != nil {
		return nil, err
	}

	log.Printf("[Server] Global version allocated: %d (DELETE)", newVersion)

	// 4️⃣ Apply soft delete
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

	// 5️⃣ Record mutation
	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO mutation_log (
			mutation_id,
			item_id,
			mutation_type,
			applied_version
		)
		VALUES ($1, $2, 'delete', $3)
		`,
		mutationID,
		id,
		newVersion,
	)
	if err != nil {
		return nil, err
	}

	deletedItem, err := r.GetByIdTx(ctx, tx, userID, id)
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

func (r *ItemRepository) getAppliedVersion(ctx context.Context, tx *sql.Tx, mutationID string) (int, bool, error) {
	var v int
	err := tx.QueryRowContext(
		ctx,
		`SELECT applied_version 
		FROM mutation_log 
		WHERE mutation_id = $1`,
		mutationID,
	).Scan(&v)

	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return v, true, nil

}
