package action

import (
	"context"
	"fmt"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
)

// ListActionsUseCase handles listing actions
type ListActionsUseCase struct {
	actionRepo repository.ActionRepository
}

// NewListActionsUseCase creates a new ListActionsUseCase
func NewListActionsUseCase(actionRepo repository.ActionRepository) *ListActionsUseCase {
	return &ListActionsUseCase{
		actionRepo: actionRepo,
	}
}

// ListActionsRequest represents the request to list actions
type ListActionsRequest struct {
	Scope       *valueobject.ActionScope
	ScopeID     string
	EnabledOnly bool
	TriggerType *entity.TriggerType
}

// Execute lists actions based on filters
func (uc *ListActionsUseCase) Execute(ctx context.Context, req ListActionsRequest) ([]*entity.Action, error) {
	var actions []*entity.Action
	var err error

	// Apply filters
	if req.TriggerType != nil {
		actions, err = uc.actionRepo.ListByTriggerType(ctx, *req.TriggerType)
		if err != nil {
			return nil, fmt.Errorf("failed to list actions by trigger type: %w", err)
		}
	} else if req.Scope != nil {
		actions, err = uc.actionRepo.ListByScope(ctx, *req.Scope, req.ScopeID)
		if err != nil {
			return nil, fmt.Errorf("failed to list actions by scope: %w", err)
		}
	} else if req.EnabledOnly {
		actions, err = uc.actionRepo.ListEnabled(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list enabled actions: %w", err)
		}
	} else {
		actions, err = uc.actionRepo.ListAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list all actions: %w", err)
		}
	}

	// Apply enabled filter if specified and not already filtered
	if req.EnabledOnly && req.TriggerType != nil {
		enabled := make([]*entity.Action, 0)
		for _, action := range actions {
			if action.Enabled() {
				enabled = append(enabled, action)
			}
		}
		actions = enabled
	}

	return actions, nil
}
