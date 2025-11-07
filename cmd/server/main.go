package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/rl-arena/rl-arena-backend/docs" // Swagger docs
	"github.com/rl-arena/rl-arena-backend/internal/api"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
	"github.com/rl-arena/rl-arena-backend/pkg/logger"
	"github.com/rl-arena/rl-arena-backend/pkg/ratelimit"
)

// @title RL-Arena API
// @version 1.0
// @description REST API server for RL-Arena - A competitive reinforcement learning platform with ELO-based rankings
// @termsOfService http://swagger.io/terms/

// @contact.name RL-Arena Support
// @contact.url https://github.com/rl-arena/rl-arena-backend/issues
// @contact.email support@rl-arena.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 로거 초기화
	logger.Init(cfg.LogLevel)
	defer logger.Sync()

	logger.Info("Starting RL-Arena Backend",
		"port", cfg.Port,
		"env", cfg.Env,
	)

	// DEBUG: DATABASE_URL 출력
	logger.Info("Database URL", "url", cfg.DatabaseURL)

	// 데이터베이스 연결
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	logger.Info("Database connection established")

	// Redis Rate Limiter 초기화 (선택적)
	var redisLimiter *ratelimit.RedisRateLimiter
	if cfg.RedisURL != "" && cfg.RedisURL != "redis://localhost:6379" {
		// Redis URL에서 주소 추출 (간단한 파싱)
		// redis://localhost:6379 -> localhost:6379
		addr := cfg.RedisURL
		if len(addr) > 8 && addr[:8] == "redis://" {
			addr = addr[8:]
		}
		
		redisLimiter = ratelimit.NewRedisRateLimiter(ratelimit.RedisRateLimiterConfig{
			Addr:         addr,
			Password:     "",
			DB:           0,
			KeyPrefix:    "ratelimit:",
			DefaultLimit: 60,
			DefaultTTL:   time.Minute,
		})
		
		// Redis 연결 확인
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		
		if err := redisLimiter.Ping(ctx); err != nil {
			logger.Warn("Failed to connect to Redis, falling back to in-memory rate limiting",
				"error", err,
				"redis_url", cfg.RedisURL,
			)
			redisLimiter.Close()
			redisLimiter = nil
		} else {
			logger.Info("Redis rate limiter initialized", "redis_url", cfg.RedisURL)
			defer redisLimiter.Close()
		}
	} else {
		logger.Info("Redis URL not configured, using in-memory rate limiting")
	}

	// 라우터 설정 (DB와 Redis Limiter 전달)
	router := api.SetupRouter(cfg, db, redisLimiter)

	// 서버 설정
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 서버 시작 (고루틴)
	go func() {
		logger.Info("Server listening", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Graceful shutdown 대기
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 10초 타임아웃으로 종료
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}
