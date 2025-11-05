package repository

import (
	"database/sql"
	"fmt"

	"github.com/rl-arena/rl-arena-backend/internal/models"
	"github.com/rl-arena/rl-arena-backend/pkg/database"
)

type UserRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 새 사용자 생성
func (r *UserRepository) Create(username, email, passwordHash, fullName string) (*models.User, error) {
	query := `
		INSERT INTO users (username, email, password_hash, full_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, full_name, avatar_url, created_at, updated_at
	`

	user := &models.User{}
	err := r.db.QueryRow(query, username, email, passwordHash, fullName).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// FindByEmail 이메일로 사용자 찾기
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 사용자 없음
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByID ID로 사용자 찾기
func (r *UserRepository) FindByID(id string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// FindByUsername 사용자명으로 찾기
func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}

// Update 사용자 정보 업데이트
func (r *UserRepository) Update(id string, fullName string, avatarURL *string) error {
	query := `
		UPDATE users
		SET full_name = $1, avatar_url = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(query, fullName, avatarURL, id)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete 사용자 삭제
func (r *UserRepository) Delete(id string) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
