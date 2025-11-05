package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Storage struct {
	basePath string
}

// NewStorage 스토리지 생성
func NewStorage(basePath string) *Storage {
	return &Storage{
		basePath: basePath,
	}
}

// SaveFile 파일 저장
func (s *Storage) SaveFile(file *multipart.FileHeader) (string, error) {
	// 파일 확장자 확인
	ext := filepath.Ext(file.Filename)
	if ext != ".py" && ext != ".zip" {
		return "", fmt.Errorf("invalid file type: only .py and .zip allowed")
	}

	// 고유 파일명 생성
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)

	// 저장 경로
	savePath := filepath.Join(s.basePath, "submissions", filename)

	// 디렉토리 생성
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 파일 열기
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// 파일 저장
	dst, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// 상대 경로 반환 (또는 URL)
	return filepath.Join("submissions", filename), nil
}

// ValidatePythonFile Python 파일 간단 검증
func (s *Storage) ValidatePythonFile(filePath string) error {
	// 파일 읽기
	content, err := os.ReadFile(filepath.Join(s.basePath, filePath))
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)

	// 기본 검증
	if !strings.Contains(contentStr, "def") {
		return fmt.Errorf("no function definitions found in Python file")
	}

	// 악성 코드 간단 체크 (더 정교한 검증 필요)
	dangerous := []string{"import os", "import sys", "subprocess", "eval(", "exec("}
	for _, danger := range dangerous {
		if strings.Contains(contentStr, danger) {
			return fmt.Errorf("potentially dangerous code detected: %s", danger)
		}
	}

	return nil
}

// DeleteFile 파일 삭제
func (s *Storage) DeleteFile(filePath string) error {
	fullPath := filepath.Join(s.basePath, filePath)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetFileURL 파일 URL 생성
func (s *Storage) GetFileURL(filePath string) string {
	// 프로덕션에서는 S3 URL 등을 반환
	return fmt.Sprintf("/storage/%s", filePath)
}
