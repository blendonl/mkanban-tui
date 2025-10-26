package action

import (
	"context"
	"fmt"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
)

// UpdateActionUseCase handles updating existing actions
type UpdateActionUseCase struct {
	actionRepo repository.ActionRepository
}

// NewUpdateActionUseCase creates a new UpdateActionUseCase
func NewUpdateActionUseCase(actionRepo repository.ActionRepository) *UpdateActionUseCase {
	return &UpdateActionUseCase{
		actionRepo: actionRepo,
	}
}

// UpdateActionRequest represents the request to update an action
type UpdateActionRequest struct {
	Name        *string
	Description *string
	Trigger     entity.Trigger
	ActionType  entity.ActionType
	Conditions  *entity.ConditionGroup
}

// Execute updates an existing action
func (uc *UpdateActionUseCase) Execute(ctx context.Context, actionID string, req UpdateActionRequest) (*entity.Action, error) {
	// Retrieve existing action
	action, err := uc.actionRepo.GetByID(ctx, actionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve action: %w", err)
	}

	// Apply updates
	if req.Name != nil {
		if err := action.UpdateName(*req.Name); err != nil {
			return nil, err
		}
	}

	if req.Description != nil {
		action.UpdateDescription(*req.Description)
	}

	if req.Trigger != nil {
		if err := action.UpdateTrigger(req.Trigger); err != nil {
			return nil, err
		}
	}

	if req.ActionType != nil {
		if err := action.UpdateActionType(req.ActionType); err != nil {
			return nil, err
		}
	}

	if req.Conditions != nil {
		action.UpdateConditions(req.Conditions)
	}

	// Persist updates
	if err := uc.actionRepo.Update(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to persist action updates: %w", err)
	}

	return action, nil
}
