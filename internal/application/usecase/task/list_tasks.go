package task

import (
	"context"
	"path/filepath"

	"mkanban/internal/application/dto"
	"mkanban/internal/domain/repository"
	"mkanban/internal/infrastructure/config"
)

// ListTasksUseCase handles listing all tasks for a board
type ListTasksUseCase struct {
	boardRepo repository.BoardRepository
	config    *config.Config
}

// NewListTasksUseCase creates a new ListTasksUseCase
func NewListTasksUseCase(boardRepo repository.BoardRepository, cfg *config.Config) *ListTasksUseCase {
	return &ListTasksUseCase{
		boardRepo: boardRepo,
		config:    cfg,
	}
}

// Execute lists all tasks for a given board with their file paths
func (uc *ListTasksUseCase) Execute(ctx context.Context, boardID string) ([]dto.TaskDTO, error) {
	board, err := uc.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.TaskDTO, 0)
	boardsPath := uc.config.Storage.BoardsPath

	// Iterate through all columns and their tasks
	for _, column := range board.Columns() {
		for _, task := range column.Tasks() {
			// Build the file path: {boardsPath}/{boardID}/{columnName}/{taskID}/task.md
			taskFolderName := task.ID().String()
			filePath := filepath.Join(boardsPath, boardID, column.Name(), taskFolderName, "task.md")

			// Convert to DTO with path and column name
			taskDTO := dto.TaskToDTOWithPath(task, filePath, column.Name())
			result = append(result, taskDTO)
		}
	}

	return result, nil
}
