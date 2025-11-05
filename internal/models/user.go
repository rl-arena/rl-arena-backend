package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // JSON에서 숨김
	FullName     string    `json:"fullName" db:"full_name"`
	AvatarURL    *string   `json:"avatarUrl,omitempty" db:"avatar_url"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type UpdateUserRequest struct {
	FullName  string `json:"fullName"`
	AvatarURL string `json:"avatarUrl"`
}

// HashPassword 비밀번호 해싱
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 비밀번호 검증
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
