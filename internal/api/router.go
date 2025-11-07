package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rl-arena/rl-arena-backend/internal/api/handlers"
	"github.com/rl-arena/rl-arena-backend/internal/api/middleware"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	"github.com/rl-arena/rl-arena-backend/internal/repository"
	"github.com/rl-arena/rl-arena-backend/internal/service"
	"github.com/rl-arena/rl-arena-backend/internal/websocket"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
	"github.com/rl-arena/rl-arena-backend/pkg/executor"
	"github.com/rl-arena/rl-arena-backend/pkg/storage"
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
	matchmakingService := service.NewMatchmakingService(
		matchmakingRepo,
		agentRepo,
		matchService,
		30*time.Second, // 30초마다 매칭 시도
	)
	matchmakingService.Start()
	println("MatchmakingService started (30s interval)")

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
	matchHandler := handlers.NewMatchHandler(matchService)
	leaderboardHandler := handlers.NewLeaderboardHandler(agentService)
	wsHandler := handlers.NewWebSocketHandler(wsHub)

	// Health check
	router.GET("/health", handlers.HealthCheck)

	// 정적 파일 서빙
	router.Static("/storage", cfg.StoragePath)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// WebSocket endpoint
		v1.GET("/ws", middleware.Auth(cfg), wsHandler.HandleWebSocket)

		// Auth routes
		auth := v1.Group("/auth")
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
			agents.POST("", middleware.Auth(cfg), agentHandler.CreateAgent)
			agents.PUT("/:id", middleware.Auth(cfg), agentHandler.UpdateAgent)
			agents.DELETE("/:id", middleware.Auth(cfg), agentHandler.DeleteAgent)
		}

		// Submission routes
		submissions := v1.Group("/submissions")
		{
			submissions.POST("", middleware.Auth(cfg), submissionHandler.CreateSubmission)
			submissions.GET("/:id", submissionHandler.GetSubmission)
			submissions.GET("/agent/:agentId", submissionHandler.ListSubmissionsByAgent)
			submissions.PUT("/:id/activate", middleware.Auth(cfg), submissionHandler.SetActiveSubmission)
			
			// 빌드 관련 엔드포인트
			submissions.GET("/:id/build-status", submissionHandler.GetBuildStatus)
			submissions.GET("/:id/build-logs", submissionHandler.GetBuildLogs)
			
			// 재빌드 엔드포인트
			submissions.POST("/:id/rebuild", middleware.Auth(cfg), submissionHandler.RebuildSubmission)
		}

		// Match routes
		matches := v1.Group("/matches")
		{
			matches.POST("", middleware.Auth(cfg), matchHandler.CreateMatch) // 수동 매치 생성
			matches.GET("", handlers.ListMatches)
			matches.GET("/:id", matchHandler.GetMatch)
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
