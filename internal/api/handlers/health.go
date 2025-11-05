package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck 서버 상태 확인
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "rl-arena-backend",
	})
}
