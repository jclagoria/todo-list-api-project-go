package middleware

import (
	"strings"

	"todo-list-api/internal/auth"
	"todo-list-api/internal/handler"

	"github.com/gin-gonic/gin"
)

func JWT(svc *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if token := strings.TrimSpace(parts[1]); token != "" {
				if userID, err := svc.ValidateAccessToken(token); err == nil {
					c.Set("userID", userID)
					c.Next()
					return
				}
			}
		}
		handler.RespondUnauthorized(c, "Missing or invalid token")
		c.Abort()
	}
}
