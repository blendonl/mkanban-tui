package action

import (
	"context"
	"fmt"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
)

// CreateActionUseCase handles creating new actions
type CreateActionUseCase struct {
	actionRepo repository.ActionRepository
}

// NewCreateActionUseCase creates a new CreateActionUseCase
func NewCreateActionUseCase(actionRepo repository.ActionRepository) *CreateActionUseCase {
	return &CreateActionUseCase{
		actionRepo: actionRepo,
	}
}

// CreateActionRequest represents the request to create an action
type CreateActionRequest struct {
	ID          string
	Name        string
	Description string
	Scope       valueobject.ActionScope
	ScopeID     string
	Trigger     entity.Trigger
	ActionType  entity.ActionType
	Conditions  *entity.ConditionGroup
}

// Execute creates a new action
func (uc *CreateActionUseCase) Execute(ctx context.Context, req CreateActionRequest) (*entity.Action, error) {
	// Validate request
	if req.ID == "" {
		return nil, entity.ErrInvalidActionID
	}
	if req.Name == "" {
		return nil, entity.ErrInvalidActionName
	}
	if !req.Scope.IsValid() {
		return nil, entity.ErrInvalidActionScope
	}

	// Create action entity
	action, err := entity.NewAction(
		req.ID,
		req.Name,
		req.Description,
		req.Scope,
		req.ScopeID,
		req.Trigger,
		req.ActionType,
		req.Conditions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create action entity: %w", err)
	}

	// Persist action
	if err := uc.actionRepo.Create(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to persist action: %w", err)
	}

	return action, nil
}
