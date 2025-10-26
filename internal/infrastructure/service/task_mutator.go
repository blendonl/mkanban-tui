package service

import (
	"context"

	"mkanban/internal/application/dto"
	"mkanban/internal/application/usecase/task"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
)

// TaskMutatorService implements the TaskMutator interface
type TaskMutatorService struct {
	createTaskUseCase *task.CreateTaskUseCase
	updateTaskUseCase *task.UpdateTaskUseCase
	moveTaskUseCase   *task.MoveTaskUseCase
}

// NewTaskMutatorService creates a new TaskMutatorService
func NewTaskMutatorService(
	createTaskUseCase *task.CreateTaskUseCase,
	updateTaskUseCase *task.UpdateTaskUseCase,
	moveTaskUseCase *task.MoveTaskUseCase,
) entity.TaskMutator {
	return &TaskMutatorService{
		createTaskUseCase: createTaskUseCase,
		updateTaskUseCase: updateTaskUseCase,
		moveTaskUseCase:   moveTaskUseCase,
	}
}

// UpdateTask updates an existing task
func (s *TaskMutatorService) UpdateTask(boardID string, taskEntity *entity.Task) error {
	ctx := context.Background()

	// Convert entity to update request
	updateReq := dto.UpdateTaskRequest{
		Title:       taskEntity.Title(),
		Description: taskEntity.Description(),
		Priority:    taskEntity.Priority().String(),
		Status:      taskEntity.Status().String(),
		Tags:        taskEntity.Tags(),
	}

	_, err := s.updateTaskUseCase.Execute(ctx, boardID, taskEntity.ID().String(), updateReq)
	return err
}

// MoveTask moves a task to another column
func (s *TaskMutatorService) MoveTask(boardID string, taskID *valueobject.TaskID, targetColumn string) error {
	ctx := context.Background()

	moveReq := dto.MoveTaskRequest{
		TaskID:           taskID.String(),
		TargetColumnName: targetColumn,
	}

	_, err := s.moveTaskUseCase.Execute(ctx, boardID, moveReq)
	return err
}

// CreateTask creates a new task
func (s *TaskMutatorService) CreateTask(boardID string, columnName string, taskEntity *entity.Task) error {
	ctx := context.Background()

	createReq := dto.CreateTaskRequest{
		Title:       taskEntity.Title(),
		Description: taskEntity.Description(),
		Priority:    taskEntity.Priority().String(),
		Status:      taskEntity.Status().String(),
		ColumnName:  columnName,
		Tags:        taskEntity.Tags(),
	}

	_, err := s.createTaskUseCase.Execute(ctx, boardID, createReq)
	return err
}
