package dto

import "time"

// BoardDTO represents a board data transfer object
type BoardDTO struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Prefix      string      `json:"prefix"`
	Description string      `json:"description"`
	Columns     []ColumnDTO `json:"columns"`
	CreatedAt   time.Time   `json:"created_at"`
	ModifiedAt  time.Time   `json:"modified_at"`
}

// CreateBoardRequest represents a request to create a board
type CreateBoardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateBoardRequest represents a request to update a board
type UpdateBoardRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// BoardListDTO represents a simplified board for listing
type BoardListDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TaskCount   int       `json:"task_count"`
	ColumnCount int       `json:"column_count"`
	ModifiedAt  time.Time `json:"modified_at"`
}
