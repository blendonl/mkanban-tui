package service

import (
	"context"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"regexp"
	"strings"
)

var (
	// nameRegex allows alphanumeric, spaces, hyphens, and underscores
	nameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s_-]+$`)
)

// ValidationService provides domain validation logic
type ValidationService struct {
	boardRepo repository.BoardRepository
}

// NewValidationService creates a new ValidationService
func NewValidationService(boardRepo repository.BoardRepository) *ValidationService {
	return &ValidationService{
		boardRepo: boardRepo,
	}
}

// ValidateBoardName validates a board name
func (v *ValidationService) ValidateBoardName(ctx context.Context, name string) error {
	// Check if empty
	if strings.TrimSpace(name) == "" {
		return entity.ErrEmptyBoardName
	}

	// Check length (3-50 characters)
	if len(name) < 3 || len(name) > 50 {
		return entity.ErrInvalidBoardName
	}

	// Check format
	if !nameRegex.MatchString(name) {
		return entity.ErrInvalidBoardName
	}

	return nil
}

// ValidateColumnName validates a column name
func (v *ValidationService) ValidateColumnName(name string) error {
	// Check if empty
	if strings.TrimSpace(name) == "" {
		return entity.ErrEmptyColumnName
	}

	// Check length (1-30 characters)
	if len(name) > 30 {
		return entity.ErrInvalidColumnName
	}

	// Check format
	if !nameRegex.MatchString(name) {
		return entity.ErrInvalidColumnName
	}

	return nil
}

// ValidateTaskTitle validates a task title
func (v *ValidationService) ValidateTaskTitle(title string) error {
	// Check if empty
	if strings.TrimSpace(title) == "" {
		return entity.ErrEmptyTaskName
	}

	// Check length (1-100 characters)
	if len(title) > 100 {
		return entity.ErrInvalidTaskName
	}

	return nil
}

// ValidateUniqueBoardName checks if a board name is unique
func (v *ValidationService) ValidateUniqueBoardName(ctx context.Context, projectID string, name string, excludeID string) error {
	existingBoard, err := v.boardRepo.FindByName(ctx, projectID, name)
	if err == nil && existingBoard != nil && existingBoard.ID() != excludeID {
		return entity.ErrBoardAlreadyExists
	}
	return nil
}

// ValidateUniqueColumnName checks if a column name is unique within a board
func (v *ValidationService) ValidateUniqueColumnName(board *entity.Board, columnName string, excludeColumn *entity.Column) error {
	for _, col := range board.Columns() {
		// Compare display names for uniqueness check
		if col.DisplayName() == columnName && col != excludeColumn {
			return entity.ErrColumnAlreadyExists
		}
	}
	return nil
}

// ValidateWIPLimit validates work-in-progress limit
func (v *ValidationService) ValidateWIPLimit(limit int) error {
	if limit < 0 {
		return entity.ErrInvalidWIPLimit
	}
	return nil
}
