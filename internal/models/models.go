package models

import "time"

type User struct {
	ID        int64
	APIKey    string
	Name      string
	CreatedAt time.Time
}
