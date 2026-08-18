package storage

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"go-api-gateway/internal/models"
)

type Store struct {
	db *sql.DB
}

func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) GetUserByAPIKey(ctx context.Context, key string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, api_key, name, created_at FROM users WHERE api_key = $1`,
		key).Scan(&u.ID, &u.APIKey, &u.Name, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
