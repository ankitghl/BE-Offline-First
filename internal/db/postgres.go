package db

import (
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

/*
**
dsn = Data Source Name
example = postgres://user:password@localhost:5432/knowledge_vault?sslmode=disable
**
*/
func NewPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
