package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"todo-list-api/internal/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrEmailTaken    = errors.New("email already taken")
	ErrTokenNotFound = errors.New("refresh token not found")
	ErrTokenExpired  = errors.New("refresh token expired")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(name, email, password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}

	id := uuid.New().String()
	_, err = r.db.Exec(
		"INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		id, name, strings.ToLower(email), string(hash),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return "", ErrEmailTaken
		}
		return "", err
	}
	return id, nil
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		"SELECT id, name, email, password_hash, created_at, updated_at, deleted_at FROM users WHERE email = ? AND deleted_at IS NULL",
		strings.ToLower(email),
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByID(id string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRow(
		"SELECT id, name, email, password_hash, created_at, updated_at, deleted_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Store(userID string) (*model.RefreshToken, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(b)

	id := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour)

	_, err := r.db.Exec(
		"INSERT INTO refresh_tokens (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)",
		id, userID, token, expiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &model.RefreshToken{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func (r *RefreshTokenRepository) FindByToken(token string) (*model.RefreshToken, error) {
	t := &model.RefreshToken{}
	err := r.db.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at, revoked_at FROM refresh_tokens WHERE token = ? AND revoked_at IS NULL",
		token,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.CreatedAt, &t.RevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(t.ExpiresAt) {
		return t, ErrTokenExpired
	}
	return t, nil
}

func (r *RefreshTokenRepository) RevokeByID(id string) error {
	_, err := r.db.Exec(
		"UPDATE refresh_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL",
		time.Now(), id,
	)
	return err
}
