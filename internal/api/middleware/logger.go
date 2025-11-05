package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/pkg/logger"
)

// Logger HTTP 요청 로깅 미들웨어
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		logger.Info("HTTP Request",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", c.Writer.Status(),
			"latency", latency,
			"ip", c.ClientIP(),
		)
	}
}
