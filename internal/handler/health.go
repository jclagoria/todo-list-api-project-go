package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterHealthRoutes(router *gin.Engine, dbConn *sql.DB) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/health/ready", func(c *gin.Context) {
		if err := dbConn.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
