package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	jwtutil "github.com/rl-arena/rl-arena-backend/pkg/jwt"
)

// Auth JWT 인증 미들웨어
func Auth(cfg *config.Config) gin.HandlerFunc {
	jwtManager := jwtutil.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)

	return func(c *gin.Context) {
		// Authorization 헤더에서 토큰 추출
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// "Bearer <token>" 형식 파싱
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// 토큰 검증
		claims, err := jwtManager.Verify(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// 검증 성공 - 사용자 정보를 context에 저장
		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}
