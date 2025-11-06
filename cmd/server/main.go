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

	"github.com/rl-arena/rl-arena-backend/internal/api"
	"github.com/rl-arena/rl-arena-backend/internal/config"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
	"github.com/rl-arena/rl-arena-backend/pkg/logger"
)

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

	// 데이터베이스 연결
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	logger.Info("Database connection established")

	// 라우터 설정 (DB 전달)
	router := api.SetupRouter(cfg, db)

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
