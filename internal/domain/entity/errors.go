package entity

import "errors"

var (
	// Board errors
	ErrBoardNotFound      = errors.New("board not found")
	ErrBoardAlreadyExists = errors.New("board already exists")
	ErrInvalidBoardName   = errors.New("invalid board name")
	ErrEmptyBoardName     = errors.New("board name cannot be empty")

	// Column errors
	ErrColumnNotFound      = errors.New("column not found")
	ErrColumnAlreadyExists = errors.New("column already exists")
	ErrInvalidColumnName   = errors.New("invalid column name")
	ErrEmptyColumnName     = errors.New("column name cannot be empty")
	ErrWIPLimitExceeded    = errors.New("work-in-progress limit exceeded")
	ErrInvalidWIPLimit     = errors.New("wip limit must be positive")

	// Task errors
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrInvalidTaskName   = errors.New("invalid task name")
	ErrEmptyTaskName     = errors.New("task name cannot be empty")
	ErrInvalidTaskID     = errors.New("invalid task ID format")

	// Session errors
	ErrSessionNotFound    = errors.New("session not found")
	ErrEmptySessionName   = errors.New("session name cannot be empty")
	ErrInvalidSessionType = errors.New("invalid session type")
	ErrEmptyWorkingDir    = errors.New("working directory cannot be empty")

	// Validation errors
	ErrInvalidPriority   = errors.New("invalid priority value")
	ErrInvalidStatus     = errors.New("invalid status value")
	ErrInvalidColor      = errors.New("invalid color format")
	ErrInvalidDate       = errors.New("invalid date")
	ErrRequiredField     = errors.New("required field is missing")
	ErrDueDateInPast     = errors.New("due date cannot be in the past")
)
