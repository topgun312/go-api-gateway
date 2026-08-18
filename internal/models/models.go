// Package models содержит структуры данных, используемые API-шлюзом.
package models

import "time"

// User — пользователь API-шлюза, идентифицируемый уникальным API-ключом.
type User struct {
	ID        int64     `json:"id"`
	APIKey    string    `json:"api_key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
