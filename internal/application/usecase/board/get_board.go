package board

import (
	"context"
	"mkanban/internal/application/dto"
	"mkanban/internal/domain/repository"
)

// GetBoardUseCase handles retrieving a single board
type GetBoardUseCase struct {
	boardRepo repository.BoardRepository
}

// NewGetBoardUseCase creates a new GetBoardUseCase
func NewGetBoardUseCase(boardRepo repository.BoardRepository) *GetBoardUseCase {
	return &GetBoardUseCase{
		boardRepo: boardRepo,
	}
}

// Execute retrieves a board by ID
func (uc *GetBoardUseCase) Execute(ctx context.Context, boardID string) (*dto.BoardDTO, error) {
	board, err := uc.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, err
	}

	boardDTO := dto.BoardToDTO(board)
	return &boardDTO, nil
}
