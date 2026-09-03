package db

import (
	"database/sql"
	"fmt"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id            TEXT PRIMARY KEY,
		name          VARCHAR(255) NOT NULL,
		email         VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at    TIMESTAMP NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL`,

	`CREATE TABLE IF NOT EXISTS todos (
		id          TEXT PRIMARY KEY,
		user_id     TEXT NOT NULL REFERENCES users(id),
		title       VARCHAR(255) NOT NULL,
		description TEXT,
		status      VARCHAR(20) NOT NULL DEFAULT 'pending'
					CHECK (status IN ('pending', 'in_progress', 'done')),
		priority    VARCHAR(20) NOT NULL DEFAULT 'medium'
					CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
		due_date    DATE,
		created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at  TIMESTAMP NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(user_id, status) WHERE deleted_at IS NULL`,

	`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id         TEXT PRIMARY KEY,
		user_id    TEXT NOT NULL REFERENCES users(id),
		token      VARCHAR(255) NOT NULL UNIQUE,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		revoked_at TIMESTAMP NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id) WHERE revoked_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token) WHERE revoked_at IS NULL`,
}

func RunMigrations(db *sql.DB) error {
	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}
	return nil
}
