package task

import (
	"context"
	"mkanban/internal/application/dto"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
)

// CreateTaskUseCase handles task creation
type CreateTaskUseCase struct {
	boardService *service.BoardService
}

// NewCreateTaskUseCase creates a new CreateTaskUseCase
func NewCreateTaskUseCase(boardService *service.BoardService) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		boardService: boardService,
	}
}

// Execute creates a new task
func (uc *CreateTaskUseCase) Execute(ctx context.Context, boardID string, req dto.CreateTaskRequest) (*dto.TaskDTO, error) {
	// Parse priority
	priority, err := valueobject.ParsePriority(req.Priority)
	if err != nil {
		priority = valueobject.PriorityNone
	}

	// Create task
	_, task, err := uc.boardService.CreateTask(
		ctx,
		boardID,
		req.ColumnName,
		req.Title,
		req.Description,
		priority,
	)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.DueDate != nil {
		_ = task.SetDueDate(*req.DueDate)
	}

	for _, tag := range req.Tags {
		task.AddTag(tag)
	}

	taskDTO := dto.TaskToDTO(task)
	return &taskDTO, nil
}
