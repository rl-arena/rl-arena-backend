package api

import (
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/rl-arena/rl-arena-backend/internal/api/handlers"
	"github.com/rl-arena/rl-arena-backend/internal/api/middleware"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/internal/service"
	"github.com/rl-arena/rl-arena-backend/internal/websocket"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
	"github.com/rl-arena/rl-arena-backend/pkg/executor"
	"github.com/rl-arena/rl-arena-backend/pkg/ratelimit"
	"github.com/rl-arena/rl-arena-backend/pkg/storage"
)

// SetupRouter API 라우터 설정
func SetupRouter(cfg *config.Config, db *database.DB, redisLimiter *ratelimit.RedisRateLimiter) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 전역 미들웨어
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// Storage 초기화
	storageManager := storage.NewStorage(cfg.StoragePath)

	// Executor 클라이언트 초기화 (gRPC)
	executorClient, err := executor.NewClient(cfg.ExecutorURL)
	if err != nil {
		panic("Failed to connect to executor: " + err.Error())
	}

	// Repository 초기화
	userRepo := repository.NewUserRepository(db)
	agentRepo := repository.NewAgentRepository(db)
	submissionRepo := repository.NewSubmissionRepository(db)
	matchRepo := repository.NewMatchRepository(db)

	// Service 초기화
	eloService := service.NewELOService()
	userService := service.NewUserService(userRepo)
	agentService := service.NewAgentService(agentRepo)

	// WebSocket Hub 초기화 및 시작
	wsHub := websocket.NewHub()
	go wsHub.Run()
	println("WebSocket Hub started")

	// Builder Service 초기화 (K8s 환경에서만)
	var builderService *service.BuilderService
	if cfg.UseK8s {
		builderService, err = service.NewBuilderService(
			submissionRepo,
			cfg.K8sNamespace,
			cfg.ContainerRegistryURL,
			cfg.ContainerRegistrySecret,
		)
		if err != nil {
			// K8s 환경이 아니거나 설정 오류 시 경고만 출력하고 계속 진행
			println("Warning: Failed to initialize BuilderService:", err.Error())
		}
	}

	submissionService := service.NewSubmissionService(submissionRepo, agentRepo, storageManager, builderService)
	matchService := service.NewMatchService(matchRepo, agentRepo, submissionRepo, eloService, executorClient)

	// Matchmaking Service 초기화 및 시작
	matchmakingRepo := repository.NewMatchmakingRepository(db)
	var matchmakingService *service.MatchmakingService
	
	if redisLimiter != nil {
		// Redis 기반 분산 Matchmaking (프로덕션)
		matchmakingService = service.NewMatchmakingServiceWithRedis(
			matchmakingRepo,
			agentRepo,
			matchService,
			30*time.Second, // 30초마다 매칭 트리거
			redisLimiter.GetClient(), // Rate Limiter의 Redis 클라이언트 재사용
		)
		println("MatchmakingService started (Redis distributed mode, 30s interval)")
	} else {
		// 단일 고루틴 Matchmaking (개발)
		matchmakingService = service.NewMatchmakingService(
			matchmakingRepo,
			agentRepo,
			matchService,
			30*time.Second,
		)
		println("MatchmakingService started (single-instance mode, 30s interval)")
	}
	
	matchmakingService.Start()

	// BuildMonitor 초기화 및 시작 (K8s 환경에서만)
	if cfg.UseK8s && builderService != nil {
		buildMonitor := service.NewBuildMonitor(
			builderService,
			submissionRepo,
			wsHub,
			matchmakingService, // 자동 매칭 큐 등록을 위해 전달
			10*time.Second,
		)
		buildMonitor.Start()
		println("BuildMonitor started with K8s Watch API + Auto-Matchmaking")
	}

	// Handler 초기화
	authHandler := handlers.NewAuthHandler(userService, cfg)
	userHandler := handlers.NewUserHandler(userService)
	agentHandler := handlers.NewAgentHandler(agentService)
	submissionHandler := handlers.NewSubmissionHandler(submissionService)
	matchHandler := handlers.NewMatchHandler(matchService, storageManager)
	leaderboardHandler := handlers.NewLeaderboardHandler(agentService)
	wsHandler := handlers.NewWebSocketHandler(wsHub)

	// Health check
	router.GET("/health", handlers.HealthCheck)

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 정적 파일 서빙
	router.Static("/storage", cfg.StoragePath)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// WebSocket endpoint
		v1.GET("/ws", middleware.Auth(cfg), wsHandler.HandleWebSocket)

		// Auth routes (rate limited by IP)
		auth := v1.Group("/auth")
		// Redis 기반 또는 In-Memory Rate Limiting 선택
		if redisLimiter != nil {
			auth.Use(middleware.RedisAuthRateLimit(redisLimiter))
		} else {
			auth.Use(middleware.AuthRateLimit())
		}
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
		}

		// Agent routes
		agents := v1.Group("/agents")
		{
			agents.GET("", agentHandler.ListAgents)
			agents.GET("/my", middleware.Auth(cfg), agentHandler.GetMyAgents)
			agents.GET("/:id", agentHandler.GetAgent)
			agents.GET("/:id/stats", agentHandler.GetAgentStats) // 상대별 전적 통계
			agents.POST("", middleware.Auth(cfg), agentHandler.CreateAgent)
			agents.PUT("/:id", middleware.Auth(cfg), agentHandler.UpdateAgent)
			agents.DELETE("/:id", middleware.Auth(cfg), agentHandler.DeleteAgent)
		}

		// Submission routes (rate limited)
		submissions := v1.Group("/submissions")
		{
			// Strict rate limit for submission creation (5 per minute per user)
			var submissionRateLimit gin.HandlerFunc
			if redisLimiter != nil {
				submissionRateLimit = middleware.RedisSubmissionRateLimit(redisLimiter)
			} else {
				submissionRateLimit = middleware.SubmissionRateLimit()
			}
			submissions.POST("", middleware.Auth(cfg), submissionRateLimit, submissionHandler.CreateSubmission)
			
			// General endpoints
			submissions.GET("/:id", submissionHandler.GetSubmission)
			submissions.GET("/agent/:agentId", submissionHandler.ListSubmissionsByAgent)
			submissions.PUT("/:id/activate", middleware.Auth(cfg), submissionHandler.SetActiveSubmission)
			
			// Build-related endpoints
			submissions.GET("/:id/build-status", submissionHandler.GetBuildStatus)
			submissions.GET("/:id/build-logs", submissionHandler.GetBuildLogs)
			
			// Rebuild endpoint (same rate limit as submission)
			submissions.POST("/:id/rebuild", middleware.Auth(cfg), submissionRateLimit, submissionHandler.RebuildSubmission)
		}

		// Match routes (rate limited)
		matches := v1.Group("/matches")
		{
			// Manual match creation (10 per minute per user)
			var matchCreationRateLimit gin.HandlerFunc
			if redisLimiter != nil {
				matchCreationRateLimit = middleware.RedisMatchCreationRateLimit(redisLimiter)
			} else {
				matchCreationRateLimit = middleware.MatchCreationRateLimit()
			}
			matches.POST("", middleware.Auth(cfg), matchCreationRateLimit, matchHandler.CreateMatch)
			
			// General match endpoints
			matches.GET("", handlers.ListMatches)
			matches.GET("/replays", matchHandler.GetMatchesWithReplays)
			matches.GET("/:id", matchHandler.GetMatch)
			
			// Replay endpoints (rate limited to prevent abuse)
			var replayDownloadRateLimit gin.HandlerFunc
			if redisLimiter != nil {
				replayDownloadRateLimit = middleware.RedisReplayDownloadRateLimit(redisLimiter)
			} else {
				replayDownloadRateLimit = middleware.ReplayDownloadRateLimit()
			}
			matches.GET("/:id/replay", replayDownloadRateLimit, matchHandler.GetMatchReplay)
			matches.GET("/:id/replay-url", matchHandler.GetMatchReplayURL)
			
			matches.GET("/agent/:agentId", matchHandler.ListMatchesByAgent)
		}

		// Leaderboard routes
		leaderboard := v1.Group("/leaderboard")
		{
			leaderboard.GET("", leaderboardHandler.GetLeaderboard)
			leaderboard.GET("/environment/:envId", leaderboardHandler.GetLeaderboardByEnvironment)
		}

		// User routes
		users := v1.Group("/users")
		users.Use(middleware.Auth(cfg))
		{
			users.GET("/me", userHandler.GetCurrentUser)
			users.PUT("/me", userHandler.UpdateCurrentUser)
		}

		// WebSocket route
		v1.GET("/ws", middleware.Auth(cfg), wsHandler.HandleWebSocket)
	}

	return router
}
