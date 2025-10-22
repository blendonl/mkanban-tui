package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/infrastructure/persistence/mapper"
	"mkanban/internal/infrastructure/serialization"
	"mkanban/pkg/filesystem"
)

// BoardRepositoryImpl implements BoardRepository using filesystem storage
type BoardRepositoryImpl struct {
	pathBuilder *PathBuilder
}

// NewBoardRepository creates a new filesystem-based board repository
func NewBoardRepository(boardsPath string) repository.BoardRepository {
	return &BoardRepositoryImpl{
		pathBuilder: NewPathBuilder(boardsPath),
	}
}

// Save persists a board to the filesystem
func (r *BoardRepositoryImpl) Save(ctx context.Context, board *entity.Board) error {
	boardDir := r.pathBuilder.BoardDir(board.ID())

	// Ensure board directory exists
	if err := filesystem.EnsureDir(boardDir, 0755); err != nil {
		return fmt.Errorf("failed to create board directory: %w", err)
	}

	// Save board metadata
	if err := r.saveBoardMetadata(board); err != nil {
		return fmt.Errorf("failed to save board metadata: %w", err)
	}

	// Save all columns
	for _, column := range board.Columns() {
		if err := r.saveColumn(board.ID(), column); err != nil {
			return fmt.Errorf("failed to save column %s: %w", column.Name(), err)
		}
	}

	// Clean up columns that no longer exist
	if err := r.cleanupOldColumns(board); err != nil {
		return fmt.Errorf("failed to cleanup old columns: %w", err)
	}

	return nil
}

// FindByID retrieves a board by its ID
func (r *BoardRepositoryImpl) FindByID(ctx context.Context, id string) (*entity.Board, error) {
	boardDir := r.pathBuilder.BoardDir(id)

	// Check if board exists
	exists, err := filesystem.Exists(boardDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, entity.ErrBoardNotFound
	}

	// Load board metadata
	board, err := r.loadBoardMetadata(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load board metadata: %w", err)
	}

	// Load all columns
	if err := r.loadColumns(board); err != nil {
		return nil, fmt.Errorf("failed to load columns: %w", err)
	}

	return board, nil
}

// FindAll retrieves all boards
func (r *BoardRepositoryImpl) FindAll(ctx context.Context) ([]*entity.Board, error) {
	rootPath := r.pathBuilder.BoardsRoot()

	// Ensure root exists
	if err := filesystem.EnsureDir(rootPath, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read boards directory: %w", err)
	}

	boards := make([]*entity.Board, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		board, err := r.FindByID(ctx, entry.Name())
		if err != nil {
			// Skip boards that can't be loaded
			continue
		}

		boards = append(boards, board)
	}

	return boards, nil
}

// Delete removes a board from storage
func (r *BoardRepositoryImpl) Delete(ctx context.Context, id string) error {
	boardDir := r.pathBuilder.BoardDir(id)

	exists, err := filesystem.Exists(boardDir)
	if err != nil {
		return err
	}
	if !exists {
		return entity.ErrBoardNotFound
	}

	return filesystem.RemoveDir(boardDir)
}

// Exists checks if a board exists
func (r *BoardRepositoryImpl) Exists(ctx context.Context, id string) (bool, error) {
	boardDir := r.pathBuilder.BoardDir(id)
	return filesystem.Exists(boardDir)
}

// FindByName finds a board by its name
func (r *BoardRepositoryImpl) FindByName(ctx context.Context, name string) (*entity.Board, error) {
	boards, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, board := range boards {
		if board.Name() == name {
			return board, nil
		}
	}

	return nil, entity.ErrBoardNotFound
}

// saveBoardMetadata saves board metadata to board.md
func (r *BoardRepositoryImpl) saveBoardMetadata(board *entity.Board) error {
	frontmatter, err := mapper.BoardToStorage(board)
	if err != nil {
		return err
	}

	data, err := serialization.SerializeFrontmatter(frontmatter, "")
	if err != nil {
		return err
	}

	metadataPath := r.pathBuilder.BoardMetadata(board.ID())
	return filesystem.SafeWrite(metadataPath, data, 0644)
}

// loadBoardMetadata loads board metadata from board.md
func (r *BoardRepositoryImpl) loadBoardMetadata(boardID string) (*entity.Board, error) {
	metadataPath := r.pathBuilder.BoardMetadata(boardID)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read board metadata: %w", err)
	}

	doc, err := serialization.ParseFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse board metadata: %w", err)
	}

	// Board name is the directory name
	boardName := boardID

	return mapper.BoardFromStorage(doc, boardName)
}

// saveColumn saves a column and all its tasks
func (r *BoardRepositoryImpl) saveColumn(boardID string, column *entity.Column) error {
	columnDir := r.pathBuilder.ColumnDir(boardID, column.Name())

	// Ensure column directory exists
	if err := filesystem.EnsureDir(columnDir, 0755); err != nil {
		return err
	}

	// Save column metadata
	frontmatter, err := mapper.ColumnToStorage(column)
	if err != nil {
		return err
	}

	data, err := serialization.SerializeFrontmatter(frontmatter, "")
	if err != nil {
		return err
	}

	metadataPath := r.pathBuilder.ColumnMetadata(boardID, column.Name())
	if err := filesystem.SafeWrite(metadataPath, data, 0644); err != nil {
		return err
	}

	// Save all tasks
	for _, task := range column.Tasks() {
		if err := r.saveTask(boardID, column.Name(), task); err != nil {
			return fmt.Errorf("failed to save task %s: %w", task.ID(), err)
		}
	}

	// Clean up tasks that no longer exist
	if err := r.cleanupOldTasks(boardID, column); err != nil {
		return fmt.Errorf("failed to cleanup old tasks: %w", err)
	}

	return nil
}

// loadColumns loads all columns for a board
func (r *BoardRepositoryImpl) loadColumns(board *entity.Board) error {
	boardDir := r.pathBuilder.BoardDir(board.ID())

	entries, err := os.ReadDir(boardDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		column, err := r.loadColumn(board.ID(), entry.Name())
		if err != nil {
			// Skip columns that can't be loaded
			continue
		}

		if err := board.AddColumn(column); err != nil {
			return err
		}
	}

	// Reorder columns based on their order field
	board.ReorderColumns()

	return nil
}

// loadColumn loads a column and its tasks
func (r *BoardRepositoryImpl) loadColumn(boardID, columnName string) (*entity.Column, error) {
	metadataPath := r.pathBuilder.ColumnMetadata(boardID, columnName)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read column metadata: %w", err)
	}

	doc, err := serialization.ParseFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse column metadata: %w", err)
	}

	column, err := mapper.ColumnFromStorage(doc, columnName)
	if err != nil {
		return nil, err
	}

	// Load all tasks
	if err := r.loadTasks(boardID, column); err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	return column, nil
}

// saveTask saves a task to filesystem
func (r *BoardRepositoryImpl) saveTask(boardID, columnName string, task *entity.Task) error {
	// Task folder name is the full task ID
	taskFolderName := task.ID().String()
	taskDir := r.pathBuilder.TaskDir(boardID, columnName, taskFolderName)

	// Ensure task directory exists
	if err := filesystem.EnsureDir(taskDir, 0755); err != nil {
		return err
	}

	// Convert task to storage format
	frontmatter, content, err := mapper.TaskToStorage(task)
	if err != nil {
		return err
	}

	data, err := serialization.SerializeFrontmatter(frontmatter, content)
	if err != nil {
		return err
	}

	metadataPath := r.pathBuilder.TaskMetadata(boardID, columnName, taskFolderName)
	return filesystem.SafeWrite(metadataPath, data, 0644)
}

// loadTasks loads all tasks for a column
func (r *BoardRepositoryImpl) loadTasks(boardID string, column *entity.Column) error {
	columnDir := r.pathBuilder.ColumnDir(boardID, column.Name())

	entries, err := os.ReadDir(columnDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		task, err := r.loadTask(boardID, column.Name(), entry.Name())
		if err != nil {
			// Skip tasks that can't be loaded
			continue
		}

		// Extract title from folder name (format: PREFIX-NUM-slug)
		// The task already has its title from the metadata
		if err := column.AddTask(task); err != nil {
			return err
		}
	}

	return nil
}

// loadTask loads a single task
func (r *BoardRepositoryImpl) loadTask(boardID, columnName, taskFolderName string) (*entity.Task, error) {
	metadataPath := r.pathBuilder.TaskMetadata(boardID, columnName, taskFolderName)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task metadata: %w", err)
	}

	doc, err := serialization.ParseFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task metadata: %w", err)
	}

	return mapper.TaskFromStorage(doc)
}

// cleanupOldColumns removes column directories that no longer exist in the board
func (r *BoardRepositoryImpl) cleanupOldColumns(board *entity.Board) error {
	boardDir := r.pathBuilder.BoardDir(board.ID())

	entries, err := os.ReadDir(boardDir)
	if err != nil {
		return err
	}

	// Get current column names
	currentColumns := make(map[string]bool)
	for _, col := range board.Columns() {
		currentColumns[col.Name()] = true
	}

	// Remove directories for columns that no longer exist
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == filepath.Base(r.pathBuilder.BoardMetadata(board.ID())) {
			continue
		}

		if !currentColumns[entry.Name()] {
			columnDir := r.pathBuilder.ColumnDir(board.ID(), entry.Name())
			if err := filesystem.RemoveDir(columnDir); err != nil {
				return err
			}
		}
	}

	return nil
}

// cleanupOldTasks removes task directories that no longer exist in the column
func (r *BoardRepositoryImpl) cleanupOldTasks(boardID string, column *entity.Column) error {
	columnDir := r.pathBuilder.ColumnDir(boardID, column.Name())

	entries, err := os.ReadDir(columnDir)
	if err != nil {
		return err
	}

	// Get current task IDs
	currentTasks := make(map[string]bool)
	for _, task := range column.Tasks() {
		currentTasks[task.ID().String()] = true
	}

	// Remove directories for tasks that no longer exist
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if !currentTasks[entry.Name()] {
			taskDir := r.pathBuilder.TaskDir(boardID, column.Name(), entry.Name())
			if err := filesystem.RemoveDir(taskDir); err != nil {
				return err
			}
		}
	}

	return nil
}
