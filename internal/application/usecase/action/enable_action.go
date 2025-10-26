package action

import (
	"context"
	"fmt"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
)

// EnableActionUseCase handles enabling/disabling actions
type EnableActionUseCase struct {
	actionRepo repository.ActionRepository
}

// NewEnableActionUseCase creates a new EnableActionUseCase
func NewEnableActionUseCase(actionRepo repository.ActionRepository) *EnableActionUseCase {
	return &EnableActionUseCase{
		actionRepo: actionRepo,
	}
}

// Execute enables an action
func (uc *EnableActionUseCase) Execute(ctx context.Context, actionID string) (*entity.Action, error) {
	action, err := uc.actionRepo.GetByID(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve action: %w", err)
	}

	action.Enable()

	if err := uc.actionRepo.Update(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to persist action: %w", err)
	}

	return action, nil
}

// DisableActionUseCase handles disabling actions
type DisableActionUseCase struct {
	actionRepo repository.ActionRepository
}

// NewDisableActionUseCase creates a new DisableActionUseCase
func NewDisableActionUseCase(actionRepo repository.ActionRepository) *DisableActionUseCase {
	return &DisableActionUseCase{
		actionRepo: actionRepo,
	}
}

// Execute disables an action
func (uc *DisableActionUseCase) Execute(ctx context.Context, actionID string) (*entity.Action, error) {
	action, err := uc.actionRepo.GetByID(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve action: %w", err)
	}

	action.Disable()

	if err := uc.actionRepo.Update(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to persist action: %w", err)
	}

	return action, nil
}
