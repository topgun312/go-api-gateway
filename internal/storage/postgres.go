// Package storage реализует хранилища приложения: PostgreSQL (пользователи)
// и клиент Redis для rate limit.
package storage

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"go-api-gateway/internal/models"
)

// Store — хранилище пользователей в PostgreSQL.
type Store struct {
	db *sql.DB
}

// NewStore открывает пул соединений с PostgreSQL по DSN и проверяет
// доступность базы (Ping). Возвращает ошибку, если подключиться нельзя.
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

// GetUserByAPIKey возвращает пользователя по API-ключу.
// Если ключ не найден, возвращает (nil, nil) — без ошибки.
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

// Close закрывает пул соединений с базой данных.
func (s *Store) Close() error {
	return s.db.Close()
}
