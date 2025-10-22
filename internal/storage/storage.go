package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mkanban/internal/model"
)

const (
	defaultDataDir  = ".local/share/mkanban"
	defaultFileName = "board.json"
)

// Storage handles persistence of board data
type Storage struct {
	dataPath string
}

// NewStorage creates a new storage instance
func NewStorage() (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, defaultDataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &Storage{
		dataPath: filepath.Join(dataDir, defaultFileName),
	}, nil
}

// LoadBoard loads the board from disk
func (s *Storage) LoadBoard() (*model.Board, error) {
	data, err := os.ReadFile(s.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default board if file doesn't exist
			return s.createDefaultBoard(), nil
		}
		return nil, fmt.Errorf("failed to read board file: %w", err)
	}

	var board model.Board
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	return &board, nil
}

// SaveBoard saves the board to disk
func (s *Storage) SaveBoard(board *model.Board) error {
	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal board: %w", err)
	}

	if err := os.WriteFile(s.dataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write board file: %w", err)
	}

	return nil
}

// createDefaultBoard creates a default board with sample data
func (s *Storage) createDefaultBoard() *model.Board {
	return &model.Board{
		Columns: []model.Column{
			{
				Title: "Todo",
				Tasks: []model.Task{
					{Title: "Design database schema", Description: ""},
					{Title: "Setup CI/CD pipeline", Description: ""},
				},
			},
			{
				Title: "In Progress",
				Tasks: []model.Task{
					{Title: "Implement authentication", Description: ""},
				},
			},
			{
				Title: "Done",
				Tasks: []model.Task{
					{Title: "Initialize project", Description: ""},
				},
			},
		},
	}
}

// GetDataPath returns the path to the data file
func (s *Storage) GetDataPath() string {
	return s.dataPath
}
