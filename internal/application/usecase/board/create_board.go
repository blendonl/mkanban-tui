package board

import (
	"context"
	"mkanban/internal/application/dto"
	"mkanban/internal/domain/service"
)

// CreateBoardUseCase handles board creation
type CreateBoardUseCase struct {
	boardService *service.BoardService
}

// NewCreateBoardUseCase creates a new CreateBoardUseCase
func NewCreateBoardUseCase(boardService *service.BoardService) *CreateBoardUseCase {
	return &CreateBoardUseCase{
		boardService: boardService,
	}
}

// Execute creates a new board
func (uc *CreateBoardUseCase) Execute(ctx context.Context, req dto.CreateBoardRequest) (*dto.BoardDTO, error) {
	board, err := uc.boardService.CreateBoard(ctx, req.ProjectID, req.Name, req.Description)
	if err != nil {
		return nil, err
	}

	boardDTO := dto.BoardToDTO(board)
	return &boardDTO, nil
}
