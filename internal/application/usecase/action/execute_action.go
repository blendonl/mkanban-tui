package action

import (
	"context"
	"fmt"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
)

// ExecuteActionUseCase handles executing a single action
type ExecuteActionUseCase struct {
	actionRepo  repository.ActionRepository
	notifier    entity.Notifier
	scriptRunner entity.ScriptRunner
	taskMutator entity.TaskMutator
}

// NewExecuteActionUseCase creates a new ExecuteActionUseCase
func NewExecuteActionUseCase(
	actionRepo repository.ActionRepository,
	notifier entity.Notifier,
	scriptRunner entity.ScriptRunner,
	taskMutator entity.TaskMutator,
) *ExecuteActionUseCase {
	return &ExecuteActionUseCase{
		actionRepo:   actionRepo,
		notifier:     notifier,
		scriptRunner: scriptRunner,
		taskMutator:  taskMutator,
	}
}

// ExecutionRequest contains the action and context to execute
type ExecutionRequest struct {
	Action        *entity.Action
	TriggerContext *entity.TriggerContext
}

// Execute executes a single action
func (uc *ExecuteActionUseCase) Execute(ctx context.Context, req ExecutionRequest) error {
	// Build action context
	actionCtx := &entity.ActionContext{
		Task:         req.TriggerContext.Task,
		Column:       req.TriggerContext.Column,
		Board:        req.TriggerContext.Board,
		Event:        req.TriggerContext.Event,
		Notifier:     uc.notifier,
		ScriptRunner: uc.scriptRunner,
		TaskMutator:  uc.taskMutator,
	}

	// Execute the action
	if err := req.Action.Execute(actionCtx); err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	// Update last run time
	if err := uc.actionRepo.UpdateLastRun(ctx, req.Action.ID()); err != nil {
		// Log error but don't fail the execution
		fmt.Printf("Failed to update last run time for action %s: %v\n", req.Action.ID(), err)
	}

	return nil
}
