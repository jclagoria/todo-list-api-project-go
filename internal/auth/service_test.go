package auth

import (
	"testing"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	svc, err := NewAuthService(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	token, err := svc.GenerateAccessToken("user-123")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	userID, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("expected user-123, got %s", userID)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	svc, err := NewAuthService(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Manually create an expired token
	claims := newExpiredClaims("user-123")
	token := signTestToken(t, svc.secret, claims)

	_, err = svc.ValidateAccessToken(token)
	if err != ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestValidateInvalidSignature(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	svc, err := NewAuthService(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Sign with a different secret
	claims := newValidClaims("user-123")
	token := signTestToken(t, []byte("wrong-secret"), claims)

	_, err = svc.ValidateAccessToken(token)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateMalformedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	svc, err := NewAuthService(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ValidateAccessToken("not-a-jwt")
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthServiceRequiresJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	_, err := NewAuthService(nil, nil)
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is empty")
	}
}
