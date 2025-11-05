package api

import (
	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/api/handlers"
	"github.com/rl-arena/rl-arena-backend/internal/api/middleware"
	"github.com/rl-arena/rl-arena-backend/internal/config"
)

// SetupRouter API 라우터 설정
func SetupRouter(cfg *config.Config) *gin.Engine {
	// Production 모드 설정
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 전역 미들웨어 설정
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// Health check
	router.GET("/health", handlers.HealthCheck)

	// API v1 그룹
	v1 := router.Group("/api/v1")
	{
		// Auth routes (인증 불필요)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", handlers.Login(cfg))
			auth.POST("/register", handlers.Register(cfg))
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
			users.GET("/me", handlers.GetCurrentUser)
			users.PUT("/me", handlers.UpdateCurrentUser)
		}
	}

	return router
}
