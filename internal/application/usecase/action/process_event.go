package action

import (
	"context"
	"fmt"
	"time"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
)

// ProcessEventUseCase handles processing domain events and triggering actions
type ProcessEventUseCase struct {
	evaluateUseCase *EvaluateActionsUseCase
	executeUseCase  *ExecuteActionUseCase
	actionRepo      repository.ActionRepository
}

// NewProcessEventUseCase creates a new ProcessEventUseCase
func NewProcessEventUseCase(
	evaluateUseCase *EvaluateActionsUseCase,
	executeUseCase *ExecuteActionUseCase,
	actionRepo repository.ActionRepository,
) *ProcessEventUseCase {
	return &ProcessEventUseCase{
		evaluateUseCase: evaluateUseCase,
		executeUseCase:  executeUseCase,
		actionRepo:      actionRepo,
	}
}

// Execute processes a domain event and executes triggered actions
func (uc *ProcessEventUseCase) Execute(ctx context.Context, event *entity.DomainEvent) error {
	// Build evaluation context from event
	evalCtx := EvaluationContext{
		BoardID:     event.BoardID,
		ColumnID:    event.ColumnID,
		Event:       event,
		CurrentTime: time.Now(),
	}

	if event.TaskID != nil {
		evalCtx.TaskID = event.TaskID.String()
	}

	// Evaluate which actions should trigger
	results, err := uc.evaluateUseCase.Execute(ctx, evalCtx)
	if err != nil {
		return fmt.Errorf("failed to evaluate actions: %w", err)
	}

	// Execute triggered actions
	for _, result := range results {
		execReq := ExecutionRequest{
			Action:         result.Action,
			TriggerContext: result.Context,
		}

		if err := uc.executeUseCase.Execute(ctx, execReq); err != nil {
			// Log error but continue with other actions
			fmt.Printf("Failed to execute action %s: %v\n", result.Action.Name(), err)
		}
	}

	return nil
}
