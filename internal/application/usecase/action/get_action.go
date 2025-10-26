package action

import (
	"context"
	"fmt"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
)

// GetActionUseCase handles retrieving a single action
type GetActionUseCase struct {
	actionRepo repository.ActionRepository
}

// NewGetActionUseCase creates a new GetActionUseCase
func NewGetActionUseCase(actionRepo repository.ActionRepository) *GetActionUseCase {
	return &GetActionUseCase{
		actionRepo: actionRepo,
	}
}

// Execute retrieves an action by ID
func (uc *GetActionUseCase) Execute(ctx context.Context, actionID string) (*entity.Action, error) {
	action, err := uc.actionRepo.GetByID(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve action: %w", err)
	}

	return action, nil
}
