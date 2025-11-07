package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
// @Summary Health check
// @Description Check if the API server is running
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Server is healthy"
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "rl-arena-backend",
	})
}
