package handler

import (
	"errors"
	"net/http"
	"time"

	"todo-list-api/internal/auth"
	"todo-list-api/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type userDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type tokenPairDTO struct {
	User         userDTO `json:"user"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
}

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type tokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func toUserDTO(u *model.User) userDTO {
	return userDTO{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

func validationDetails(err error) []ErrorDetail {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := make([]ErrorDetail, 0, len(ve))
		for _, fe := range ve {
			details = append(details, ErrorDetail{Field: fe.Field(), Message: fe.Tag()})
		}
		return details
	}
	return []ErrorDetail{{Field: "body", Message: "invalid request body"}}
}

func isTokenError(err error) bool {
	return errors.Is(err, auth.ErrInvalidToken) ||
		errors.Is(err, auth.ErrExpiredToken) ||
		errors.Is(err, auth.ErrRevokedToken)
}

func respondTokenError(c *gin.Context, err error) {
	if isTokenError(err) {
		RespondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	RespondInternalError(c)
}

func RegisterAuthRoutes(router *gin.Engine, svc *auth.AuthService) {
	router.POST("/v1/register", func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			RespondValidationError(c, validationDetails(err))
			return
		}

		user, access, refresh, err := svc.Register(req.Name, req.Email, req.Password)
		if err != nil {
			if errors.Is(err, auth.ErrEmailTaken) {
				RespondConflict(c, "email already taken")
				return
			}
			RespondInternalError(c)
			return
		}

		RespondCreated(c, tokenPairDTO{
			User:         toUserDTO(user),
			AccessToken:  access,
			RefreshToken: refresh,
		})
	})

	router.POST("/v1/login", func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			RespondValidationError(c, validationDetails(err))
			return
		}

		user, access, refresh, err := svc.Login(req.Email, req.Password)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				RespondUnauthorized(c, "invalid credentials")
				return
			}
			RespondInternalError(c)
			return
		}

		RespondSuccess(c, tokenPairDTO{
			User:         toUserDTO(user),
			AccessToken:  access,
			RefreshToken: refresh,
		})
	})

	router.POST("/v1/refresh", func(c *gin.Context) {
		var req tokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			RespondValidationError(c, validationDetails(err))
			return
		}

		user, access, refresh, err := svc.Refresh(req.RefreshToken)
		if err != nil {
			respondTokenError(c, err)
			return
		}

		RespondSuccess(c, tokenPairDTO{
			User:         toUserDTO(user),
			AccessToken:  access,
			RefreshToken: refresh,
		})
	})

	router.POST("/v1/logout", func(c *gin.Context) {
		var req tokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			RespondValidationError(c, validationDetails(err))
			return
		}

		if err := svc.RevokeRefreshToken(req.RefreshToken); err != nil {
			respondTokenError(c, err)
			return
		}

		RespondSuccess(c, gin.H{"message": "logged out"})
	})
}
