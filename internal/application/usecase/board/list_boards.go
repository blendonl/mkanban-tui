package board

import (
	"context"
	"mkanban/internal/application/dto"
	"mkanban/internal/domain/repository"
)

// ListBoardsUseCase handles listing all boards
type ListBoardsUseCase struct {
	boardRepo repository.BoardRepository
}

// NewListBoardsUseCase creates a new ListBoardsUseCase
func NewListBoardsUseCase(boardRepo repository.BoardRepository) *ListBoardsUseCase {
	return &ListBoardsUseCase{
		boardRepo: boardRepo,
	}
}

// Execute lists all boards
func (uc *ListBoardsUseCase) Execute(ctx context.Context) ([]dto.BoardListDTO, error) {
	boards, err := uc.boardRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.BoardListDTO, 0, len(boards))
	for _, board := range boards {
		result = append(result, dto.BoardToListDTO(board))
	}

	return result, nil
}
