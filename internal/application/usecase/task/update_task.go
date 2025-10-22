package task

import (
	"context"
	"mkanban/internal/application/dto"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
)

// UpdateTaskUseCase handles task updates
type UpdateTaskUseCase struct {
	boardService *service.BoardService
}

// NewUpdateTaskUseCase creates a new UpdateTaskUseCase
func NewUpdateTaskUseCase(boardService *service.BoardService) *UpdateTaskUseCase {
	return &UpdateTaskUseCase{
		boardService: boardService,
	}
}

// Execute updates a task
func (uc *UpdateTaskUseCase) Execute(ctx context.Context, boardID string, taskIDStr string, req dto.UpdateTaskRequest) (*dto.TaskDTO, error) {
	// Parse task ID
	taskID, err := valueobject.ParseTaskID(taskIDStr)
	if err != nil {
		return nil, err
	}

	// Parse optional priority
	var priority *valueobject.Priority
	if req.Priority != nil {
		p, err := valueobject.ParsePriority(*req.Priority)
		if err == nil {
			priority = &p
		}
	}

	// Parse optional status
	var status *valueobject.Status
	if req.Status != nil {
		s, err := valueobject.ParseStatus(*req.Status)
		if err == nil {
			status = &s
		}
	}

	// Update task
	_, task, err := uc.boardService.UpdateTask(
		ctx,
		boardID,
		taskID,
		req.Title,
		req.Description,
		priority,
		status,
	)
	if err != nil {
		return nil, err
	}

	// Update optional fields
	if req.DueDate != nil {
		_ = task.SetDueDate(*req.DueDate)
	}

	taskDTO := dto.TaskToDTO(task)
	return &taskDTO, nil
}
