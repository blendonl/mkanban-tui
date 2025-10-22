package task

import (
	"context"
	"mkanban/internal/application/dto"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
)

// MoveTaskUseCase handles moving tasks between columns
type MoveTaskUseCase struct {
	boardService *service.BoardService
}

// NewMoveTaskUseCase creates a new MoveTaskUseCase
func NewMoveTaskUseCase(boardService *service.BoardService) *MoveTaskUseCase {
	return &MoveTaskUseCase{
		boardService: boardService,
	}
}

// Execute moves a task to a different column
func (uc *MoveTaskUseCase) Execute(ctx context.Context, boardID string, req dto.MoveTaskRequest) (*dto.BoardDTO, error) {
	// Parse task ID
	taskID, err := valueobject.ParseTaskID(req.TaskID)
	if err != nil {
		return nil, err
	}

	// Move task
	board, err := uc.boardService.MoveTask(ctx, boardID, taskID, req.TargetColumnName)
	if err != nil {
		return nil, err
	}

	boardDTO := dto.BoardToDTO(board)
	return &boardDTO, nil
}
