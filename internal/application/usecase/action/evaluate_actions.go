package action

import (
	"context"
	"time"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
)

// EvaluateActionsUseCase handles evaluating which actions should trigger
type EvaluateActionsUseCase struct {
	actionRepo  repository.ActionRepository
	boardRepo   repository.BoardRepository
}

// NewEvaluateActionsUseCase creates a new EvaluateActionsUseCase
func NewEvaluateActionsUseCase(
	actionRepo repository.ActionRepository,
	boardRepo repository.BoardRepository,
) *EvaluateActionsUseCase {
	return &EvaluateActionsUseCase{
		actionRepo: actionRepo,
		boardRepo:  boardRepo,
	}
}

// EvaluationContext contains information for evaluating actions
type EvaluationContext struct {
	BoardID     string
	ColumnID    string
	TaskID      string
	Event       *entity.DomainEvent
	CurrentTime time.Time
}

// EvaluationResult represents the result of evaluating actions
type EvaluationResult struct {
	Action  *entity.Action
	Context *entity.TriggerContext
}

// Execute evaluates actions and returns those that should trigger
func (uc *EvaluateActionsUseCase) Execute(ctx context.Context, evalCtx EvaluationContext) ([]*EvaluationResult, error) {
	results := make([]*EvaluationResult, 0)

	// Get all enabled actions
	actions, err := uc.actionRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	// Get board, column, task context if needed
	var board *entity.Board
	var column *entity.Column
	var task *entity.Task

	if evalCtx.BoardID != "" {
		board, err = uc.boardRepo.GetByID(ctx, evalCtx.BoardID)
		if err != nil {
			// Continue without board context
			board = nil
		}
	}

	if board != nil && evalCtx.ColumnID != "" {
		column, _ = board.GetColumn(evalCtx.ColumnID)
	}

	if board != nil && evalCtx.TaskID != "" {
		task, column, _ = board.FindTask(nil) // TODO: Need TaskID parsing
	}

	// Evaluate each action
	for _, action := range actions {
		// Check if action scope matches
		if !action.MatchesScope(evalCtx.BoardID, evalCtx.ColumnID, evalCtx.TaskID) {
			continue
		}

		// Build trigger context
		triggerCtx := &entity.TriggerContext{
			CurrentTime: evalCtx.CurrentTime,
			Task:        task,
			Column:      column,
			Board:       board,
			Event:       evalCtx.Event,
			LastRun:     action.LastRun(),
		}

		// Check if action should execute
		if action.ShouldExecute(triggerCtx) {
			results = append(results, &EvaluationResult{
				Action:  action,
				Context: triggerCtx,
			})
		}
	}

	return results, nil
}
