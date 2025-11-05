package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/rl-arena/rl-arena-backend/pkg/logger"
)

type DB struct {
	*sql.DB
}

// Connect 데이터베이스 연결
func Connect(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is empty")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 연결 풀 설정
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 연결 테스트
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connected successfully")

	return &DB{db}, nil
}

// Close 데이터베이스 연결 종료
func (db *DB) Close() error {
	return db.DB.Close()
}
