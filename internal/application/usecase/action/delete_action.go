package action

import (
	"context"
	"fmt"

	"mkanban/internal/domain/repository"
)

// DeleteActionUseCase handles deleting actions
type DeleteActionUseCase struct {
	actionRepo repository.ActionRepository
}

// NewDeleteActionUseCase creates a new DeleteActionUseCase
func NewDeleteActionUseCase(actionRepo repository.ActionRepository) *DeleteActionUseCase {
	return &DeleteActionUseCase{
		actionRepo: actionRepo,
	}
}

// Execute deletes an action by ID
func (uc *DeleteActionUseCase) Execute(ctx context.Context, actionID string) error {
	if err := uc.actionRepo.Delete(ctx, actionID); err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}

	return nil
}
