package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/infrastructure/serialization"
	"time"
)

// BoardStorage represents board storage format
type BoardStorage struct {
	ID          string    `yaml:"id"`
	Prefix      string    `yaml:"prefix"`
	Created     time.Time `yaml:"created"`
	Modified    time.Time `yaml:"modified"`
	Description string    `yaml:"description"`
	NextTaskNum int       `yaml:"next_task_num"`
}

// BoardToStorage converts a Board entity to storage format
func BoardToStorage(board *entity.Board) (map[string]interface{}, error) {
	frontmatter := map[string]interface{}{
		"id":            board.ID(),
		"prefix":        board.Prefix(),
		"created":       board.CreatedAt().Format(time.RFC3339),
		"modified":      board.ModifiedAt().Format(time.RFC3339),
		"description":   board.Description(),
		"next_task_num": board.NextTaskNum(),
	}

	return frontmatter, nil
}

// BoardFromStorage converts storage format to Board entity
func BoardFromStorage(doc *serialization.FrontmatterDocument, name string) (*entity.Board, error) {
	id := doc.GetString("id")
	if id == "" {
		return nil, fmt.Errorf("missing board ID")
	}

	description := doc.GetString("description")

	board, err := entity.NewBoard(id, name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create board: %w", err)
	}

	// Set next task number
	nextTaskNum := doc.GetInt("next_task_num")
	if nextTaskNum > 0 {
		board.SetNextTaskNum(nextTaskNum)
	}

	return board, nil
}
