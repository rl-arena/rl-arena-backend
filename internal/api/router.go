package api

import (
	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/api/handlers"
	"github.com/rl-arena/rl-arena-backend/internal/api/middleware"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/internal/service"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
)

// SetupRouter API 라우터 설정
func SetupRouter(cfg *config.Config, db *database.DB) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 전역 미들웨어
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// Repository 초기화
	userRepo := repository.NewUserRepository(db)

	// Service 초기화
	userService := service.NewUserService(userRepo)

	// Handler 초기화
	authHandler := handlers.NewAuthHandler(userService, cfg)
	userHandler := handlers.NewUserHandler(userService)

	// Health check
	router.GET("/health", handlers.HealthCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Auth routes (인증 불필요)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
		}

		// Agent routes
		agents := v1.Group("/agents")
		{
			agents.GET("", handlers.ListAgents)
			agents.GET("/:id", handlers.GetAgent)
			agents.POST("", middleware.Auth(cfg), handlers.CreateAgent)
			agents.PUT("/:id", middleware.Auth(cfg), handlers.UpdateAgent)
			agents.DELETE("/:id", middleware.Auth(cfg), handlers.DeleteAgent)
		}

		// Submission routes
		submissions := v1.Group("/submissions")
		{
			submissions.POST("", middleware.Auth(cfg), handlers.CreateSubmission)
			submissions.GET("/:id", handlers.GetSubmission)
			submissions.GET("/agent/:agentId", handlers.ListSubmissionsByAgent)
		}

		// Match routes
		matches := v1.Group("/matches")
		{
			matches.GET("", handlers.ListMatches)
			matches.GET("/:id", handlers.GetMatch)
			matches.GET("/agent/:agentId", handlers.ListMatchesByAgent)
		}

		// Leaderboard routes
		leaderboard := v1.Group("/leaderboard")
		{
			leaderboard.GET("", handlers.GetLeaderboard)
			leaderboard.GET("/environment/:envId", handlers.GetLeaderboardByEnvironment)
		}

		// User routes (모두 인증 필요)
		users := v1.Group("/users")
		users.Use(middleware.Auth(cfg))
		{
			users.GET("/me", userHandler.GetCurrentUser)
			users.PUT("/me", userHandler.UpdateCurrentUser)
		}
	}

	return router
}
