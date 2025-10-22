package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

// Board is the aggregate root for the kanban board domain
type Board struct {
	id          string
	name        string
	prefix      string
	description string
	columns     []*Column
	nextTaskNum int
	createdAt   time.Time
	modifiedAt  time.Time
}

// NewBoard creates a new Board entity (aggregate root)
func NewBoard(id string, name string, description string) (*Board, error) {
	if id == "" {
		return nil, ErrInvalidBoardName
	}
	if name == "" {
		return nil, ErrEmptyBoardName
	}

	prefix := valueobject.GenerateBoardPrefix(name)
	now := time.Now()

	return &Board{
		id:          id,
		name:        name,
		prefix:      prefix,
		description: description,
		columns:     make([]*Column, 0),
		nextTaskNum: 1,
		createdAt:   now,
		modifiedAt:  now,
	}, nil
}

// ID returns the board ID
func (b *Board) ID() string {
	return b.id
}

// Name returns the board name
func (b *Board) Name() string {
	return b.name
}

// Prefix returns the board prefix for task IDs
func (b *Board) Prefix() string {
	return b.prefix
}

// Description returns the board description
func (b *Board) Description() string {
	return b.description
}

// Columns returns a copy of the columns
func (b *Board) Columns() []*Column {
	columnsCopy := make([]*Column, len(b.columns))
	copy(columnsCopy, b.columns)
	return columnsCopy
}

// NextTaskNum returns the next task number
func (b *Board) NextTaskNum() int {
	return b.nextTaskNum
}

// CreatedAt returns when the board was created
func (b *Board) CreatedAt() time.Time {
	return b.createdAt
}

// ModifiedAt returns when the board was last modified
func (b *Board) ModifiedAt() time.Time {
	return b.modifiedAt
}

// UpdateName updates the board name and regenerates prefix
func (b *Board) UpdateName(name string) error {
	if name == "" {
		return ErrEmptyBoardName
	}
	b.name = name
	b.prefix = valueobject.GenerateBoardPrefix(name)
	b.modifiedAt = time.Now()
	return nil
}

// UpdateDescription updates the board description
func (b *Board) UpdateDescription(description string) {
	b.description = description
	b.modifiedAt = time.Now()
}

// AddColumn adds a column to the board
func (b *Board) AddColumn(column *Column) error {
	if column == nil {
		return ErrColumnNotFound
	}

	// Check if column with same name already exists
	for _, existingColumn := range b.columns {
		if existingColumn.Name() == column.Name() {
			return ErrColumnAlreadyExists
		}
	}

	b.columns = append(b.columns, column)
	b.modifiedAt = time.Now()
	return nil
}

// RemoveColumn removes a column from the board
func (b *Board) RemoveColumn(columnName string) (*Column, error) {
	for i, column := range b.columns {
		if column.Name() == columnName {
			// Check if column has tasks
			if column.TaskCount() > 0 {
				return nil, ErrColumnNotFound // Could create a specific error for this
			}

			// Remove column from slice
			b.columns = append(b.columns[:i], b.columns[i+1:]...)
			b.modifiedAt = time.Now()
			return column, nil
		}
	}
	return nil, ErrColumnNotFound
}

// GetColumn retrieves a column by name
func (b *Board) GetColumn(columnName string) (*Column, error) {
	for _, column := range b.columns {
		if column.Name() == columnName {
			return column, nil
		}
	}
	return nil, ErrColumnNotFound
}

// GetColumnByIndex retrieves a column by its index
func (b *Board) GetColumnByIndex(index int) (*Column, error) {
	if index < 0 || index >= len(b.columns) {
		return nil, ErrColumnNotFound
	}
	return b.columns[index], nil
}

// FindTask finds a task across all columns
func (b *Board) FindTask(taskID *valueobject.TaskID) (*Task, *Column, error) {
	for _, column := range b.columns {
		if task, err := column.GetTask(taskID); err == nil {
			return task, column, nil
		}
	}
	return nil, nil, ErrTaskNotFound
}

// MoveTask moves a task from one column to another
func (b *Board) MoveTask(taskID *valueobject.TaskID, targetColumnName string) error {
	// Find the task and its current column
	task, sourceColumn, err := b.FindTask(taskID)
	if err != nil {
		return err
	}

	// Get target column
	targetColumn, err := b.GetColumn(targetColumnName)
	if err != nil {
		return err
	}

	// Check if target column can accept the task (WIP limit)
	if !targetColumn.CanAddTask() {
		return ErrWIPLimitExceeded
	}

	// Remove from source column
	if _, err := sourceColumn.RemoveTask(taskID); err != nil {
		return err
	}

	// Add to target column
	if err := targetColumn.AddTask(task); err != nil {
		// Rollback: add back to source column
		_ = sourceColumn.AddTask(task)
		return err
	}

	b.modifiedAt = time.Now()
	return nil
}

// GenerateNextTaskID generates the next task ID for this board
func (b *Board) GenerateNextTaskID(slug string) (*valueobject.TaskID, error) {
	taskID, err := valueobject.NewTaskID(b.prefix, b.nextTaskNum, slug)
	if err != nil {
		return nil, err
	}
	b.nextTaskNum++
	b.modifiedAt = time.Now()
	return taskID, nil
}

// SetNextTaskNum sets the next task number (used during loading from storage)
func (b *Board) SetNextTaskNum(num int) {
	if num > b.nextTaskNum {
		b.nextTaskNum = num
	}
}

// ColumnCount returns the number of columns
func (b *Board) ColumnCount() int {
	return len(b.columns)
}

// TotalTaskCount returns the total number of tasks across all columns
func (b *Board) TotalTaskCount() int {
	total := 0
	for _, column := range b.columns {
		total += column.TaskCount()
	}
	return total
}

// ReorderColumns reorders the columns based on their order field
func (b *Board) ReorderColumns() {
	// Simple bubble sort by order field
	n := len(b.columns)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if b.columns[j].Order() > b.columns[j+1].Order() {
				b.columns[j], b.columns[j+1] = b.columns[j+1], b.columns[j]
			}
		}
	}
	b.modifiedAt = time.Now()
}
