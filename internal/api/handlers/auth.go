package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	"github.com/rl-arena/rl-arena-backend/internal/service"
	jwtutil "github.com/rl-arena/rl-arena-backend/pkg/jwt"
	"github.com/rl-arena/rl-arena-backend/pkg/logger"
)

type AuthHandler struct {
	userService *service.UserService
	jwtManager  *jwtutil.JWTManager
}

func NewAuthHandler(userService *service.UserService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtManager:  jwtutil.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration),
	}
}

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
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 사용자 인증
	user, err := h.userService.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to login",
		})
		return
	}

	// JWT 토큰 생성
	token, err := h.jwtManager.Generate(user.ID, user.Username, user.Email)
	if err != nil {
		logger.Error("Failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	logger.Info("User logged in", "userId", user.ID, "email", user.Email)

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User: gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"fullName":  user.FullName,
			"avatarUrl": user.AvatarURL,
		},
	})
}

// Register 회원가입
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 사용자 생성
	user, err := h.userService.Register(req.Username, req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "User already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register user",
		})
		return
	}

	// JWT 토큰 생성
	token, err := h.jwtManager.Generate(user.ID, user.Username, user.Email)
	if err != nil {
		logger.Error("Failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	logger.Info("User registered", "userId", user.ID, "email", user.Email)

	c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User: gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"fullName":  user.FullName,
			"avatarUrl": user.AvatarURL,
		},
	})
}
