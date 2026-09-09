package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newValidClaims(userID string) *jwt.RegisteredClaims {
	now := time.Now()
	return &jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
	}
}

func newExpiredClaims(userID string) *jwt.RegisteredClaims {
	now := time.Now()
	return &jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-30 * time.Minute)),
		ExpiresAt: jwt.NewNumericDate(now.Add(-15 * time.Minute)),
	}
}

func signTestToken(t *testing.T, secret []byte, claims *jwt.RegisteredClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("signTestToken failed: %v", err)
	}
	return s
}
