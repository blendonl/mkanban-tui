package repository

import (
	"context"
	"mkanban/internal/domain/entity"
)

// BoardRepository defines the interface for board persistence
type BoardRepository interface {
	// Save persists a board to storage
	Save(ctx context.Context, board *entity.Board) error

	// FindByID retrieves a board by its ID
	FindByID(ctx context.Context, id string) (*entity.Board, error)

	// FindAll retrieves all boards
	FindAll(ctx context.Context) ([]*entity.Board, error)

	// Delete removes a board from storage
	Delete(ctx context.Context, id string) error

	// Exists checks if a board exists
	Exists(ctx context.Context, id string) (bool, error)

	// FindByName finds a board by its name
	FindByName(ctx context.Context, name string) (*entity.Board, error)
}
