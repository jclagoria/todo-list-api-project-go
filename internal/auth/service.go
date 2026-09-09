package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrExpiredToken  = errors.New("token expired")
	ErrRevokedToken  = errors.New("token revoked")
)

type AuthService struct {
	secret   []byte
	users    *UserRepository
	tokens   *RefreshTokenRepository
}

func NewAuthService(users *UserRepository, tokens *RefreshTokenRepository) (*AuthService, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
	}
	return &AuthService{
		secret: []byte(secret),
		users:  users,
		tokens: tokens,
	}, nil
}

func (s *AuthService) GenerateAccessToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

func (s *AuthService) CreateRefreshToken(userID string) (string, time.Time, error) {
	rt, err := s.tokens.Store(userID)
	if err != nil {
		return "", time.Time{}, err
	}
	return rt.Token, rt.ExpiresAt, nil
}

func (s *AuthService) RotateRefreshToken(currentToken string) (string, string, error) {
	rt, err := s.tokens.FindByToken(currentToken)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) {
			return "", "", ErrExpiredToken
		}
		return "", "", ErrInvalidToken
	}
	if rt == nil {
		return "", "", ErrInvalidToken
	}

	if err := s.tokens.RevokeByID(rt.ID); err != nil {
		return "", "", err
	}

	newRT, err := s.tokens.Store(rt.UserID)
	if err != nil {
		return "", "", err
	}

	newAT, err := s.GenerateAccessToken(rt.UserID)
	if err != nil {
		return "", "", err
	}

	return newAT, newRT.Token, nil
}

func (s *AuthService) RevokeRefreshToken(token string) error {
	rt, err := s.tokens.FindByToken(token)
	if err != nil {
		return ErrInvalidToken
	}
	if rt == nil {
		return ErrInvalidToken
	}
	return s.tokens.RevokeByID(rt.ID)
}
