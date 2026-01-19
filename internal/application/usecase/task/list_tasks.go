package task

import (
	"context"
	"path/filepath"

	"mkanban/internal/application/dto"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/config"
	"mkanban/pkg/filesystem"
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
	dataPath := uc.config.Storage.DataPath

	// Iterate through all columns and their tasks
	for _, column := range board.Columns() {
		for _, task := range column.Tasks() {
			taskFolderName := task.ID().String()
			filePath, err := buildTaskFilePath(dataPath, boardID, column.Name(), taskFolderName)
			if err != nil {
				return nil, err
			}

			// Convert to DTO with path and column name
			taskDTO := dto.TaskToDTOWithPath(task, filePath, column.Name())
			result = append(result, taskDTO)
		}
	}

	return result, nil
}

func buildTaskFilePath(dataPath, boardID, columnName, taskFolderName string) (string, error) {
	projectSlug, boardSlug, err := valueobject.ParseBoardID(boardID)
	if err != nil {
		return "", err
	}
	boardDir := filepath.Join(dataPath, "projects", projectSlug, "boards", boardSlug)
	columnsDir := filepath.Join(boardDir, "columns")
	columnsExists, err := filesystem.Exists(columnsDir)
	if err != nil {
		return "", err
	}

	columnRoot := boardDir
	if columnsExists {
		columnRoot = columnsDir
	}

	columnDir := filepath.Join(columnRoot, columnName)
	tasksDir := filepath.Join(columnDir, "tasks")
	tasksExists, err := filesystem.Exists(tasksDir)
	if err != nil {
		return "", err
	}

	if tasksExists {
		return filepath.Join(tasksDir, taskFolderName, "task.md"), nil
	}

	return filepath.Join(columnDir, taskFolderName, "task.md"), nil
}
