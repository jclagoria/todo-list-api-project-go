package auth

import (
	"database/sql"
	"testing"
	"time"

	"todo-list-api/internal/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := db.ConnectTest()
	if err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}
	t.Cleanup(func() { _ = testDB.Close() })
	return testDB
}

func TestUserRepository_Create(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewUserRepository(testDB)

	id, err := repo.Create("Alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	// Duplicate email
	_, err = repo.Create("Bob", "alice@example.com", "password456")
	if err != ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewUserRepository(testDB)

	_, _ = repo.Create("Alice", "alice@example.com", "password123")

	// Found
	u, err := repo.FindByEmail("alice@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if u == nil {
		t.Fatal("expected user, got nil")
	}
	if u.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", u.Name)
	}

	// Not found
	u, err = repo.FindByEmail("missing@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if u != nil {
		t.Fatal("expected nil for missing email")
	}

	// Soft-deleted
	_, err = testDB.Exec("UPDATE users SET deleted_at = CURRENT_TIMESTAMP WHERE email = ?", "alice@example.com")
	if err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}
	u, err = repo.FindByEmail("alice@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if u != nil {
		t.Fatal("expected nil for soft-deleted user")
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	testDB := setupTestDB(t)
	repo := NewUserRepository(testDB)

	id, _ := repo.Create("Alice", "alice@example.com", "password123")

	// Found
	u, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if u == nil {
		t.Fatal("expected user, got nil")
	}
	if u.Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %s", u.Email)
	}

	// Not found
	u, err = repo.FindByID("nonexistent")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if u != nil {
		t.Fatal("expected nil for nonexistent ID")
	}
}

func TestRefreshTokenRepository_StoreFindRevoke(t *testing.T) {
	testDB := setupTestDB(t)
	userRepo := NewUserRepository(testDB)
	tokenRepo := NewRefreshTokenRepository(testDB)

	userID, _ := userRepo.Create("Alice", "alice@example.com", "password123")

	// Store
	rt, err := tokenRepo.Store(userID)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	if rt.Token == "" {
		t.Fatal("expected non-empty token")
	}

	// Find
	found, err := tokenRepo.FindByToken(rt.Token)
	if err != nil {
		t.Fatalf("FindByToken failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected token, got nil")
	}
	if found.UserID != userID {
		t.Fatalf("expected user %s, got %s", userID, found.UserID)
	}

	// Revoke
	err = tokenRepo.RevokeByID(rt.ID)
	if err != nil {
		t.Fatalf("RevokeByID failed: %v", err)
	}

	// Find after revoke — should return nil
	found, err = tokenRepo.FindByToken(rt.Token)
	if err != nil {
		t.Fatalf("FindByToken failed: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for revoked token")
	}
}

func TestAuthService_RotateRefreshToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	testDB := setupTestDB(t)
	userRepo := NewUserRepository(testDB)
	tokenRepo := NewRefreshTokenRepository(testDB)
	svc, err := NewAuthService(userRepo, tokenRepo)
	if err != nil {
		t.Fatal(err)
	}

	userID, _ := userRepo.Create("Alice", "alice@example.com", "password123")
	rt, _ := tokenRepo.Store(userID)

	// Successful rotation
	newAT, newRT, err := svc.RotateRefreshToken(rt.Token)
	if err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}
	if newAT == "" {
		t.Fatal("expected non-empty access token")
	}
	if newRT == "" {
		t.Fatal("expected non-empty refresh token")
	}

	// Old token should be revoked
	found, _ := tokenRepo.FindByToken(rt.Token)
	if found != nil {
		t.Fatal("old token should be revoked")
	}

	// Rotate again with new token should work
	newAT2, newRT2, err := svc.RotateRefreshToken(newRT)
	if err != nil {
		t.Fatalf("second rotation failed: %v", err)
	}
	if newAT2 == "" || newRT2 == "" {
		t.Fatal("expected non-empty tokens from second rotation")
	}
}

func TestAuthService_RotateRevokedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	testDB := setupTestDB(t)
	userRepo := NewUserRepository(testDB)
	tokenRepo := NewRefreshTokenRepository(testDB)
	svc, err := NewAuthService(userRepo, tokenRepo)
	if err != nil {
		t.Fatal(err)
	}

	userID, _ := userRepo.Create("Alice", "alice@example.com", "password123")
	rt, _ := tokenRepo.Store(userID)
	_ = tokenRepo.RevokeByID(rt.ID)

	_, _, err = svc.RotateRefreshToken(rt.Token)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for revoked token, got %v", err)
	}
}

func TestAuthService_RotateExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	testDB := setupTestDB(t)
	userRepo := NewUserRepository(testDB)
	tokenRepo := NewRefreshTokenRepository(testDB)
	svc, err := NewAuthService(userRepo, tokenRepo)
	if err != nil {
		t.Fatal(err)
	}

	userID, _ := userRepo.Create("Alice", "alice@example.com", "password123")
	rt, _ := tokenRepo.Store(userID)

	// Backdate the token to make it expired
	_, err = testDB.Exec("UPDATE refresh_tokens SET expires_at = ? WHERE id = ?", time.Now().Add(-1*time.Hour), rt.ID)
	if err != nil {
		t.Fatalf("backdate failed: %v", err)
	}

	_, _, err = svc.RotateRefreshToken(rt.Token)
	if err != ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}
