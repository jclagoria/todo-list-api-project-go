package model

import (
	"database/sql"
	"time"
)

type RefreshToken struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Token     string       `json:"-"`
	ExpiresAt time.Time    `json:"expires_at"`
	CreatedAt time.Time    `json:"created_at"`
	RevokedAt sql.NullTime `json:"-"`
}
