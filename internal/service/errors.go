package service

import "errors"

// Common service errors
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("resource not found")
)

// User service specific errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

// Agent service specific errors
var (
	ErrAgentNotFound      = errors.New("agent not found")
	ErrInvalidEnvironment = errors.New("invalid environment")
)

// Match service specific errors
var (
	ErrMatchNotFound = errors.New("match not found")
)

// Submission service specific errors
var (
	ErrSubmissionNotFound   = errors.New("submission not found")
	ErrInvalidFile          = errors.New("invalid file")
	ErrDailyQuotaExceeded   = errors.New("daily submission quota exceeded")
)
