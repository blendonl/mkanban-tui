package column

import (
	"context"
	"mkanban/internal/application/dto"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
)

// CreateColumnUseCase handles column creation
type CreateColumnUseCase struct {
	boardService *service.BoardService
}

// NewCreateColumnUseCase creates a new CreateColumnUseCase
func NewCreateColumnUseCase(boardService *service.BoardService) *CreateColumnUseCase {
	return &CreateColumnUseCase{
		boardService: boardService,
	}
}

// Execute creates a new column
func (uc *CreateColumnUseCase) Execute(ctx context.Context, boardID string, req dto.CreateColumnRequest) (*dto.BoardDTO, error) {
	// Parse color if provided
	var color *valueobject.Color
	if req.Color != nil && *req.Color != "" {
		c, err := valueobject.NewColor(*req.Color)
		if err == nil {
			color = c
		}
	}

	// Create column
	board, err := uc.boardService.AddColumnToBoard(
		ctx,
		boardID,
		req.Name,
		req.Description,
		req.Order,
		req.WIPLimit,
		color,
	)
	if err != nil {
		return nil, err
	}

	boardDTO := dto.BoardToDTO(board)
	return &boardDTO, nil
}
