package repository

import (
	"context"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
)

// ActionRepository defines the interface for action persistence
type ActionRepository interface {
	// Create creates a new action
	Create(ctx context.Context, action *entity.Action) error

	// Update updates an existing action
	Update(ctx context.Context, action *entity.Action) error

	// Delete deletes an action by ID
	Delete(ctx context.Context, id string) error

	// GetByID retrieves an action by ID
	GetByID(ctx context.Context, id string) (*entity.Action, error)

	// ListAll retrieves all actions
	ListAll(ctx context.Context) ([]*entity.Action, error)

	// ListByScope retrieves actions for a specific scope
	ListByScope(ctx context.Context, scope valueobject.ActionScope, scopeID string) ([]*entity.Action, error)

	// ListGlobal retrieves all global actions
	ListGlobal(ctx context.Context) ([]*entity.Action, error)

	// ListByBoard retrieves actions for a specific board
	ListByBoard(ctx context.Context, boardID string) ([]*entity.Action, error)

	// ListByColumn retrieves actions for a specific column
	ListByColumn(ctx context.Context, columnID string) ([]*entity.Action, error)

	// ListByTask retrieves actions for a specific task
	ListByTask(ctx context.Context, taskID string) ([]*entity.Action, error)

	// ListEnabled retrieves all enabled actions
	ListEnabled(ctx context.Context) ([]*entity.Action, error)

	// ListByTriggerType retrieves actions by trigger type
	ListByTriggerType(ctx context.Context, triggerType entity.TriggerType) ([]*entity.Action, error)

	// UpdateLastRun updates the last run time for an action
	UpdateLastRun(ctx context.Context, id string) error
}
