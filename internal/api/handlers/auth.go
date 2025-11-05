package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	jwtutil "github.com/rl-arena/rl-arena-backend/pkg/jwt"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"fullName"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  gin.H  `json:"user"`
}

// Login 로그인
func Login(cfg *config.Config) gin.HandlerFunc {
	jwtManager := jwtutil.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)

	return func(c *gin.Context) {
		var req LoginRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// TODO: 데이터베이스에서 사용자 확인
		// user, err := userService.FindByEmail(req.Email)
		// if err != nil || !user.CheckPassword(req.Password) {
		//     c.JSON(401, gin.H{"error": "Invalid credentials"})
		//     return
		// }

		// 임시: 하드코딩된 사용자
		if req.Email != "test@example.com" || req.Password != "password123" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}

		// 토큰 생성
		token, err := jwtManager.Generate("user-123", "testuser", req.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate token",
			})
			return
		}

		c.JSON(http.StatusOK, AuthResponse{
			Token: token,
			User: gin.H{
				"id":       "user-123",
				"username": "testuser",
				"email":    req.Email,
			},
		})
	}
}

// Register 회원가입
func Register(cfg *config.Config) gin.HandlerFunc {
	jwtManager := jwtutil.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)

	return func(c *gin.Context) {
		var req RegisterRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// TODO: 데이터베이스에 사용자 생성
		// user, err := userService.Create(req)
		// if err != nil {
		//     c.JSON(400, gin.H{"error": err.Error()})
		//     return
		// }

		// 임시: 바로 토큰 생성
		userID := "user-new-123"
		token, err := jwtManager.Generate(userID, req.Username, req.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate token",
			})
			return
		}

		c.JSON(http.StatusCreated, AuthResponse{
			Token: token,
			User: gin.H{
				"id":       userID,
				"username": req.Username,
				"email":    req.Email,
				"fullName": req.FullName,
			},
		})
	}
}
