package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/infrastructure/serialization"
	"time"
)

// BoardStorage represents board metadata storage format (metadata.yml)
type BoardStorage struct {
	ID          string    `yaml:"id"`
	Prefix      string    `yaml:"prefix"`
	Created     time.Time `yaml:"created"`
	Modified    time.Time `yaml:"modified"`
	NextTaskNum int       `yaml:"next_task_num"`
}

// BoardMetadataToStorage converts a Board entity to metadata storage format
func BoardMetadataToStorage(board *entity.Board) (map[string]interface{}, error) {
	metadata := map[string]interface{}{
		"id":            board.ID(),
		"prefix":        board.Prefix(),
		"created":       board.CreatedAt().Format(time.RFC3339),
		"modified":      board.ModifiedAt().Format(time.RFC3339),
		"next_task_num": board.NextTaskNum(),
	}

	return metadata, nil
}

// BoardContentToMarkdown converts a Board entity to markdown content (board.md)
func BoardContentToMarkdown(board *entity.Board) []byte {
	return serialization.SerializeMarkdownWithTitle(board.Name(), board.Description())
}

// BoardFromStorage converts storage format to Board entity (new split format)
func BoardFromStorage(metadataDoc *serialization.FrontmatterDocument, name string, description string) (*entity.Board, error) {
	id := metadataDoc.GetString("id")
	if id == "" {
		return nil, fmt.Errorf("missing board ID")
	}

	board, err := entity.NewBoard(id, name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create board: %w", err)
	}

	// Set next task number
	nextTaskNum := metadataDoc.GetInt("next_task_num")
	if nextTaskNum > 0 {
		board.SetNextTaskNum(nextTaskNum)
	}

	return board, nil
}

// BoardFromLegacyStorage converts old frontmatter format to Board entity (backward compatibility)
func BoardFromLegacyStorage(doc *serialization.FrontmatterDocument, boardID string) (*entity.Board, error) {
	id := doc.GetString("id")
	if id == "" {
		id = boardID // Use directory name as fallback
	}

	// In legacy format, name was derived from directory, not stored
	name := boardID
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
