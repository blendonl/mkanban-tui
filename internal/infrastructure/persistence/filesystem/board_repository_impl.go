package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/persistence/mapper"
	"mkanban/internal/infrastructure/serialization"
	"mkanban/pkg/filesystem"
	"mkanban/pkg/slug"
)

// BoardRepositoryImpl implements BoardRepository using filesystem storage
type BoardRepositoryImpl struct {
	pathBuilder *PathBuilder
}

// NewBoardRepository creates a new filesystem-based board repository
func NewBoardRepository(rootPath string) repository.BoardRepository {
	return &BoardRepositoryImpl{
		pathBuilder: NewPathBuilder(rootPath),
	}
}

// Save persists a board to the filesystem
func (r *BoardRepositoryImpl) Save(ctx context.Context, board *entity.Board) error {
	boardDir, err := r.pathBuilder.BoardDir(board.ID())
	if err != nil {
		return err
	}

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
	boardDir, err := r.pathBuilder.BoardDir(id)
	if err != nil {
		return nil, err
	}

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
	projectsRoot := r.pathBuilder.projectPathBuilder.ProjectsRoot()

	if err := filesystem.EnsureDir(projectsRoot, 0755); err != nil {
		return nil, err
	}

	projectEntries, err := os.ReadDir(projectsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	boards := make([]*entity.Board, 0)
	for _, projectEntry := range projectEntries {
		if !projectEntry.IsDir() {
			continue
		}

		boardsDir := r.pathBuilder.projectPathBuilder.ProjectBoardsDir(projectEntry.Name())
		boardEntries, err := os.ReadDir(boardsDir)
		if err != nil {
			continue
		}

		for _, boardEntry := range boardEntries {
			if !boardEntry.IsDir() {
				continue
			}

			boardID, err := valueobject.BuildBoardID(projectEntry.Name(), boardEntry.Name())
			if err != nil {
				continue
			}

			board, err := r.FindByID(ctx, boardID)
			if err != nil {
				continue
			}

			boards = append(boards, board)
		}
	}

	return boards, nil
}

// Delete removes a board from storage
func (r *BoardRepositoryImpl) Delete(ctx context.Context, id string) error {
	boardDir, err := r.pathBuilder.BoardDir(id)
	if err != nil {
		return err
	}

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
	boardDir, err := r.pathBuilder.BoardDir(id)
	if err != nil {
		return false, err
	}
	return filesystem.Exists(boardDir)
}

// FindByName finds a board by its name within a project
func (r *BoardRepositoryImpl) FindByName(ctx context.Context, projectID string, name string) (*entity.Board, error) {
	boardsDir := r.pathBuilder.projectPathBuilder.ProjectBoardsDir(projectID)
	entries, err := os.ReadDir(boardsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, entity.ErrBoardNotFound
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		boardID, err := valueobject.BuildBoardID(projectID, entry.Name())
		if err != nil {
			continue
		}

		board, err := r.FindByID(ctx, boardID)
		if err != nil {
			continue
		}

		if board.Name() == name {
			return board, nil
		}
	}

	return nil, entity.ErrBoardNotFound
}

// saveBoardMetadata saves board metadata to metadata.yml and content to board.md
func (r *BoardRepositoryImpl) saveBoardMetadata(board *entity.Board) error {
	// Save metadata.yml
	metadata, err := mapper.BoardMetadataToStorage(board)
	if err != nil {
		return err
	}

	metadataYaml, err := serialization.SerializeYaml(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	metadataPath, err := r.pathBuilder.BoardMetadataYaml(board.ID())
	if err != nil {
		return err
	}
	if err := filesystem.SafeWrite(metadataPath, metadataYaml, 0644); err != nil {
		return fmt.Errorf("failed to write metadata.yml: %w", err)
	}

	// Save board.md with name and description
	markdown := mapper.BoardContentToMarkdown(board)
	contentPath, err := r.pathBuilder.BoardContent(board.ID())
	if err != nil {
		return err
	}
	if err := filesystem.SafeWrite(contentPath, markdown, 0644); err != nil {
		return fmt.Errorf("failed to write board.md: %w", err)
	}

	return nil
}

// loadBoardMetadata loads board metadata from metadata.yml and board.md (with backward compatibility)
func (r *BoardRepositoryImpl) loadBoardMetadata(boardID string) (*entity.Board, error) {
	// Try new format first: metadata.yml + board.md
	metadataYamlPath, err := r.pathBuilder.BoardMetadataYaml(boardID)
	if err != nil {
		return nil, err
	}
	contentPath, err := r.pathBuilder.BoardContent(boardID)
	if err != nil {
		return nil, err
	}

	metadataYamlData, metadataErr := os.ReadFile(metadataYamlPath)
	contentData, contentErr := os.ReadFile(contentPath)

	if metadataErr == nil && contentErr == nil {
		// New split format
		var metadataMap map[string]interface{}
		if err := serialization.ParseYaml(metadataYamlData, &metadataMap); err != nil {
			return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
		}

		// Parse board.md for name and description
		markdownDoc, err := serialization.ParseMarkdownWithTitle(contentData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse board.md: %w", err)
		}

		// Create frontmatter document for mapper compatibility
		doc := &serialization.FrontmatterDocument{
			Frontmatter: metadataMap,
		}

		board, err := mapper.BoardFromStorage(doc, markdownDoc.Title, markdownDoc.Content)
		if err != nil {
			return nil, err
		}
		if board.ProjectID() == "" {
			projectSlug, _, err := valueobject.ParseBoardID(boardID)
			if err == nil {
				board.SetProjectID(projectSlug)
			}
		}
		return board, nil
	}

	// Fallback to old format: board.md with frontmatter
	// Note: Old format didn't have board.md, it was using different file name
	// Let's check if there's a legacy board.md with frontmatter
	if contentErr == nil {
		// Try parsing as frontmatter document (old format)
		doc, err := serialization.ParseFrontmatter(contentData)
		if err == nil {
			// This is the legacy format (frontmatter in board.md)
			return mapper.BoardFromLegacyStorage(doc, boardID)
		}
	}

	// If neither format works, return an error
	return nil, fmt.Errorf("failed to load board metadata: metadata.yml error: %v, board.md error: %v", metadataErr, contentErr)
}

// saveColumn saves a column and all its tasks
func (r *BoardRepositoryImpl) saveColumn(boardID string, column *entity.Column) error {
	// Generate normalized folder name from display name if needed
	normalizedName := slug.Generate(column.DisplayName())
	columnDir, err := r.pathBuilder.ColumnDir(boardID, normalizedName)
	if err != nil {
		return err
	}

	// Ensure column directory exists
	if err := filesystem.EnsureDir(columnDir, 0755); err != nil {
		return err
	}

	// Save column metadata to metadata.yml
	metadata, err := mapper.ColumnMetadataToStorage(column)
	if err != nil {
		return err
	}

	yamlData, err := serialization.SerializeYaml(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize column metadata: %w", err)
	}

	metadataYamlPath, err := r.pathBuilder.ColumnMetadataYaml(boardID, normalizedName)
	if err != nil {
		return err
	}
	if err := filesystem.SafeWrite(metadataYamlPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write column metadata.yml: %w", err)
	}

	// Save column.md with display name and description
	markdown := mapper.ColumnContentToMarkdown(column)
	contentPath, err := r.pathBuilder.ColumnContent(boardID, normalizedName)
	if err != nil {
		return err
	}
	if err := filesystem.SafeWrite(contentPath, markdown, 0644); err != nil {
		return fmt.Errorf("failed to write column.md: %w", err)
	}

	// Save all tasks
	for _, task := range column.Tasks() {
		if err := r.saveTask(boardID, normalizedName, task); err != nil {
			return fmt.Errorf("failed to save task %s: %w", task.ID(), err)
		}
	}

	// Clean up tasks that no longer exist
	if err := r.cleanupOldTasks(boardID, normalizedName, column); err != nil {
		return fmt.Errorf("failed to cleanup old tasks: %w", err)
	}

	return nil
}

// loadColumns loads all columns for a board
func (r *BoardRepositoryImpl) loadColumns(board *entity.Board) error {
	boardDir, err := r.pathBuilder.BoardDir(board.ID())
	if err != nil {
		return err
	}
	columnsDir := filepath.Join(boardDir, "columns")

	// Check if columns directory exists (new format)
	exists, err := filesystem.Exists(columnsDir)
	if err != nil {
		return err
	}

	// If columns/ doesn't exist, try loading from board root (legacy format)
	if !exists {
		return fmt.Errorf("missing columns directory for board: %s", board.ID())
	}

	entries, err := os.ReadDir(columnsDir)
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

// loadColumnsLegacy loads columns from board root directory (legacy format)
func (r *BoardRepositoryImpl) loadColumnsLegacy(board *entity.Board) error {
	boardDir, err := r.pathBuilder.BoardDir(board.ID())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(boardDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip the columns directory if it exists
		if entry.Name() == "columns" {
			continue
		}

		// Check if this directory is a column by looking for column.md or metadata.yml
		columnDir := filepath.Join(boardDir, entry.Name())
		columnMdPath := filepath.Join(columnDir, "column.md")
		metadataYamlPath := filepath.Join(columnDir, "metadata.yml")

		hasColumnMd, _ := filesystem.Exists(columnMdPath)
		hasMetadataYaml, _ := filesystem.Exists(metadataYamlPath)

		if !hasColumnMd && !hasMetadataYaml {
			// Not a column directory
			continue
		}

		// Load column using legacy path (directly under board)
		column, err := r.loadColumnLegacy(board.ID(), entry.Name())
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
func (r *BoardRepositoryImpl) loadColumn(boardID, columnFolderName string) (*entity.Column, error) {
	// Try new format first: metadata.yml + column.md
	metadataYamlPath, err := r.pathBuilder.ColumnMetadataYaml(boardID, columnFolderName)
	if err != nil {
		return nil, err
	}
	contentPath, err := r.pathBuilder.ColumnContent(boardID, columnFolderName)
	if err != nil {
		return nil, err
	}

	metadataYamlData, metadataErr := os.ReadFile(metadataYamlPath)
	contentData, contentErr := os.ReadFile(contentPath)

	if metadataErr == nil && contentErr == nil {
		// New split format
		var metadataMap map[string]interface{}
		if err := serialization.ParseYaml(metadataYamlData, &metadataMap); err != nil {
			return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
		}

		// Parse column.md for display name and description
		markdownDoc, err := serialization.ParseMarkdownWithTitle(contentData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse column.md: %w", err)
		}

		// Create frontmatter document for mapper compatibility
		doc := &serialization.FrontmatterDocument{
			Frontmatter: metadataMap,
		}

		column, err := mapper.ColumnFromStorage(doc, columnFolderName, markdownDoc.Title, markdownDoc.Content)
		if err != nil {
			return nil, err
		}

		// Load all tasks
		if err := r.loadTasks(boardID, columnFolderName, column); err != nil {
			return nil, fmt.Errorf("failed to load tasks: %w", err)
		}

		return column, nil
	}

	// Fallback to old format with column.md containing frontmatter
	if contentErr == nil {
		// Try parsing as frontmatter document (old format)
		doc, err := serialization.ParseFrontmatter(contentData)
		if err == nil {
			// This is the legacy format (frontmatter in column.md)
			column, err := mapper.ColumnFromLegacyStorage(doc, columnFolderName)
			if err != nil {
				return nil, err
			}

			// Load all tasks
			if err := r.loadTasks(boardID, columnFolderName, column); err != nil {
				return nil, fmt.Errorf("failed to load tasks: %w", err)
			}

			return column, nil
		}
	}

	// If new format exists but only metadata.yml (no column.md yet)
	if metadataErr == nil {
		// Use metadata.yml with folder name as display name
		var metadataMap map[string]interface{}
		if err := serialization.ParseYaml(metadataYamlData, &metadataMap); err != nil {
			return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
		}

		doc := &serialization.FrontmatterDocument{
			Frontmatter: metadataMap,
		}

		// Use folder name as display name, empty description
		column, err := mapper.ColumnFromStorage(doc, columnFolderName, columnFolderName, "")
		if err != nil {
			return nil, err
		}

		// Load all tasks
		if err := r.loadTasks(boardID, columnFolderName, column); err != nil {
			return nil, fmt.Errorf("failed to load tasks: %w", err)
		}

		return column, nil
	}

	// If neither format works, return an error
	return nil, fmt.Errorf("failed to load column metadata: metadata.yml error: %v, column.md error: %v", metadataErr, contentErr)
}

// loadColumnLegacy loads a column from the legacy location (directly under board directory)
func (r *BoardRepositoryImpl) loadColumnLegacy(boardID, columnFolderName string) (*entity.Column, error) {
	boardDir, err := r.pathBuilder.BoardDir(boardID)
	if err != nil {
		return nil, err
	}

	// Try new format first: metadata.yml + column.md
	metadataYamlPath := filepath.Join(boardDir, columnFolderName, "metadata.yml")
	contentPath := filepath.Join(boardDir, columnFolderName, "column.md")

	metadataYamlData, metadataErr := os.ReadFile(metadataYamlPath)
	contentData, contentErr := os.ReadFile(contentPath)

	if metadataErr == nil && contentErr == nil {
		// New split format
		var metadataMap map[string]interface{}
		if err := serialization.ParseYaml(metadataYamlData, &metadataMap); err != nil {
			return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
		}

		// Parse column.md for display name and description
		markdownDoc, err := serialization.ParseMarkdownWithTitle(contentData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse column.md: %w", err)
		}

		// Create frontmatter document for mapper compatibility
		doc := &serialization.FrontmatterDocument{
			Frontmatter: metadataMap,
		}

		column, err := mapper.ColumnFromStorage(doc, columnFolderName, markdownDoc.Title, markdownDoc.Content)
		if err != nil {
			return nil, err
		}

		// Load all tasks using legacy paths
		if err := r.loadTasksLegacy(boardID, columnFolderName, column); err != nil {
			return nil, fmt.Errorf("failed to load tasks: %w", err)
		}

		return column, nil
	}

	// Fallback to old format with column.md containing frontmatter
	if contentErr == nil {
		// Try parsing as frontmatter document (old format)
		doc, err := serialization.ParseFrontmatter(contentData)
		if err == nil {
			// This is the legacy format (frontmatter in column.md)
			column, err := mapper.ColumnFromLegacyStorage(doc, columnFolderName)
			if err != nil {
				return nil, err
			}

			// Load all tasks using legacy paths
			if err := r.loadTasksLegacy(boardID, columnFolderName, column); err != nil {
				return nil, fmt.Errorf("failed to load tasks: %w", err)
			}

			return column, nil
		}
	}

	// If new format exists but only metadata.yml (no column.md yet)
	if metadataErr == nil {
		// Use metadata.yml with folder name as display name
		var metadataMap map[string]interface{}
		if err := serialization.ParseYaml(metadataYamlData, &metadataMap); err != nil {
			return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
		}

		doc := &serialization.FrontmatterDocument{
			Frontmatter: metadataMap,
		}

		// Use folder name as display name, empty description
		column, err := mapper.ColumnFromStorage(doc, columnFolderName, columnFolderName, "")
		if err != nil {
			return nil, err
		}

		// Load all tasks using legacy paths
		if err := r.loadTasksLegacy(boardID, columnFolderName, column); err != nil {
			return nil, fmt.Errorf("failed to load tasks: %w", err)
		}

		return column, nil
	}

	// If neither format works, return an error
	return nil, fmt.Errorf("failed to load column metadata: metadata.yml error: %v, column.md error: %v", metadataErr, contentErr)
}

// SaveTask persists a single task without rewriting the entire board
func (r *BoardRepositoryImpl) SaveTask(ctx context.Context, boardID string, columnName string, task *entity.Task) error {
	return r.saveTask(boardID, columnName, task)
}

// saveTask saves a task to filesystem
func (r *BoardRepositoryImpl) saveTask(boardID, columnName string, task *entity.Task) error {
	// Task folder name is the full task ID
	taskFolderName := task.ID().String()
	taskDir, err := r.pathBuilder.TaskDir(boardID, columnName, taskFolderName)
	if err != nil {
		return err
	}

	// Ensure task directory exists
	if err := filesystem.EnsureDir(taskDir, 0755); err != nil {
		return err
	}

	// Convert task to storage format
	storage, markdownContent, err := mapper.TaskToStorage(task)
	if err != nil {
		return err
	}

	// Save metadata.yml
	metadataYamlPath, err := r.pathBuilder.TaskMetadataYaml(boardID, columnName, taskFolderName)
	if err != nil {
		return err
	}
	metadataYamlData, err := serialization.SerializeYaml(storage)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}
	if err := filesystem.SafeWrite(metadataYamlPath, metadataYamlData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata.yml: %w", err)
	}

	// Save task.md
	taskMdPath, err := r.pathBuilder.TaskMetadata(boardID, columnName, taskFolderName)
	if err != nil {
		return err
	}
	if err := filesystem.SafeWrite(taskMdPath, markdownContent, 0644); err != nil {
		return fmt.Errorf("failed to write task.md: %w", err)
	}

	return nil
}

// loadTasks loads all tasks for a column
func (r *BoardRepositoryImpl) loadTasks(boardID string, columnFolderName string, column *entity.Column) error {
	columnDir, err := r.pathBuilder.ColumnDir(boardID, columnFolderName)
	if err != nil {
		return err
	}
	tasksDir := filepath.Join(columnDir, "tasks")

	// Check if tasks/ subdirectory exists (new format)
	exists, err := filesystem.Exists(tasksDir)
	if err != nil {
		return err
	}

	// If tasks/ doesn't exist, try loading from column root (legacy format)
	if !exists {
		return r.loadTasksFromColumnRoot(boardID, columnFolderName, column)
	}

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		task, err := r.loadTask(boardID, columnFolderName, entry.Name())
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

// loadTasksFromColumnRoot loads tasks directly from column root (legacy format without tasks/ subdirectory)
func (r *BoardRepositoryImpl) loadTasksFromColumnRoot(boardID string, columnFolderName string, column *entity.Column) error {
	columnDir, err := r.pathBuilder.ColumnDir(boardID, columnFolderName)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(columnDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip non-task directories (metadata files, column.md, etc.)
		// Tasks should have the format PREFIX-NUMBER-slug
		if !r.isTaskDirectory(entry.Name()) {
			continue
		}

		task, err := r.loadTaskFromColumnRoot(boardID, columnFolderName, entry.Name())
		if err != nil {
			// Skip tasks that can't be loaded
			continue
		}

		if err := column.AddTask(task); err != nil {
			return err
		}
	}

	return nil
}

// loadTasksLegacy loads all tasks for a column from legacy location (directly under board)
func (r *BoardRepositoryImpl) loadTasksLegacy(boardID string, columnFolderName string, column *entity.Column) error {
	boardDir, err := r.pathBuilder.BoardDir(boardID)
	if err != nil {
		return err
	}
	columnDir := filepath.Join(boardDir, columnFolderName)

	entries, err := os.ReadDir(columnDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip non-task directories
		if !r.isTaskDirectory(entry.Name()) {
			continue
		}

		task, err := r.loadTaskLegacy(boardID, columnFolderName, entry.Name())
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

// isTaskDirectory checks if a directory name matches the task folder format (PREFIX-NUMBER-slug)
func (r *BoardRepositoryImpl) isTaskDirectory(name string) bool {
	// Try to parse as TaskID - if it succeeds, it's a task directory
	_, err := valueobject.ParseTaskID(name)
	return err == nil
}

// loadTask loads a single task
func (r *BoardRepositoryImpl) loadTask(boardID, columnName, taskFolderName string) (*entity.Task, error) {
	// Read metadata.yml
	metadataYamlPath, err := r.pathBuilder.TaskMetadataYaml(boardID, columnName, taskFolderName)
	if err != nil {
		return nil, err
	}
	metadataYamlData, err := os.ReadFile(metadataYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.yml: %w", err)
	}

	// Parse metadata
	var storage mapper.TaskStorage
	if err := serialization.ParseYaml(metadataYamlData, &storage); err != nil {
		return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
	}

	// Read task.md
	taskMdPath, err := r.pathBuilder.TaskMetadata(boardID, columnName, taskFolderName)
	if err != nil {
		return nil, err
	}
	markdownData, err := os.ReadFile(taskMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task.md: %w", err)
	}

	// Parse TaskID from folder name (which has format PREFIX-NUMBER-slug)
	// This ensures we preserve the full TaskID including the slug
	taskID, err := valueobject.ParseTaskID(taskFolderName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task ID from folder name %s: %w", taskFolderName, err)
	}

	return mapper.TaskFromStorage(&storage, markdownData, taskID)
}

// loadTaskFromColumnRoot loads a single task from column root (legacy format without tasks/ subdirectory)
func (r *BoardRepositoryImpl) loadTaskFromColumnRoot(boardID, columnName, taskFolderName string) (*entity.Task, error) {
	columnDir, err := r.pathBuilder.ColumnDir(boardID, columnName)
	if err != nil {
		return nil, err
	}

	// Read metadata.yml from column root
	metadataYamlPath := filepath.Join(columnDir, taskFolderName, "metadata.yml")
	metadataYamlData, err := os.ReadFile(metadataYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.yml: %w", err)
	}

	// Parse metadata
	var storage mapper.TaskStorage
	if err := serialization.ParseYaml(metadataYamlData, &storage); err != nil {
		return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
	}

	// Read task.md from column root
	taskMdPath := filepath.Join(columnDir, taskFolderName, "task.md")
	markdownData, err := os.ReadFile(taskMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task.md: %w", err)
	}

	// Parse TaskID from folder name (which has format PREFIX-NUMBER-slug)
	taskID, err := valueobject.ParseTaskID(taskFolderName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task ID from folder name %s: %w", taskFolderName, err)
	}

	return mapper.TaskFromStorage(&storage, markdownData, taskID)
}

// loadTaskLegacy loads a single task from legacy location (directly under board)
func (r *BoardRepositoryImpl) loadTaskLegacy(boardID, columnName, taskFolderName string) (*entity.Task, error) {
	boardDir, err := r.pathBuilder.BoardDir(boardID)
	if err != nil {
		return nil, err
	}

	// Read metadata.yml from legacy location
	metadataYamlPath := filepath.Join(boardDir, columnName, taskFolderName, "metadata.yml")
	metadataYamlData, err := os.ReadFile(metadataYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.yml: %w", err)
	}

	// Parse metadata
	var storage mapper.TaskStorage
	if err := serialization.ParseYaml(metadataYamlData, &storage); err != nil {
		return nil, fmt.Errorf("failed to parse metadata.yml: %w", err)
	}

	// Read task.md from legacy location
	taskMdPath := filepath.Join(boardDir, columnName, taskFolderName, "task.md")
	markdownData, err := os.ReadFile(taskMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task.md: %w", err)
	}

	// Parse TaskID from folder name (which has format PREFIX-NUMBER-slug)
	// This ensures we preserve the full TaskID including the slug
	taskID, err := valueobject.ParseTaskID(taskFolderName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task ID from folder name %s: %w", taskFolderName, err)
	}

	return mapper.TaskFromStorage(&storage, markdownData, taskID)
}

// cleanupOldColumns removes column directories that no longer exist in the board
func (r *BoardRepositoryImpl) cleanupOldColumns(board *entity.Board) error {
	boardDir, err := r.pathBuilder.BoardDir(board.ID())
	if err != nil {
		return err
	}
	columnsDir := filepath.Join(boardDir, "columns")

	// Check if columns directory exists
	exists, err := filesystem.Exists(columnsDir)
	if err != nil {
		return err
	}

	// If columns/ doesn't exist, nothing to cleanup in new format
	if !exists {
		return nil
	}

	entries, err := os.ReadDir(columnsDir)
	if err != nil {
		return err
	}

	// Get current column folder names (normalized)
	currentColumns := make(map[string]bool)
	for _, col := range board.Columns() {
		normalizedName := slug.Generate(col.DisplayName())
		currentColumns[normalizedName] = true
	}

	// Remove directories for columns that no longer exist
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if !currentColumns[entry.Name()] {
			columnDir, err := r.pathBuilder.ColumnDir(board.ID(), entry.Name())
			if err != nil {
				return err
			}
			if err := filesystem.RemoveDir(columnDir); err != nil {
				return err
			}
		}
	}

	return nil
}

// cleanupOldTasks removes task directories that no longer exist in the column
func (r *BoardRepositoryImpl) cleanupOldTasks(boardID string, columnFolderName string, column *entity.Column) error {
	columnDir, err := r.pathBuilder.ColumnDir(boardID, columnFolderName)
	if err != nil {
		return err
	}
	tasksDir := filepath.Join(columnDir, "tasks")

	// Check if tasks/ subdirectory exists
	exists, err := filesystem.Exists(tasksDir)
	if err != nil {
		return err
	}

	// If tasks/ doesn't exist, nothing to cleanup in new format
	if !exists {
		return nil
	}

	entries, err := os.ReadDir(tasksDir)
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
			taskDir, err := r.pathBuilder.TaskDir(boardID, columnFolderName, entry.Name())
			if err != nil {
				return err
			}
			if err := filesystem.RemoveDir(taskDir); err != nil {
				return err
			}
		}
	}

	return nil
}

// MigrateColumnsToSubdirectory migrates columns from board root to columns/ subdirectory
func (r *BoardRepositoryImpl) MigrateColumnsToSubdirectory(ctx context.Context, boardID string) error {
	boardDir, err := r.pathBuilder.BoardDir(boardID)
	if err != nil {
		return err
	}
	columnsDir := filepath.Join(boardDir, "columns")

	// Check if columns/ already exists
	columnsExists, err := filesystem.Exists(columnsDir)
	if err != nil {
		return fmt.Errorf("failed to check columns directory: %w", err)
	}

	// If columns/ already exists, assume migration is done
	if columnsExists {
		return nil
	}

	entries, err := os.ReadDir(boardDir)
	if err != nil {
		return fmt.Errorf("failed to read board directory: %w", err)
	}

	// Find all column directories (those with column.md or metadata.yml)
	columnsToMigrate := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip non-column directories
		if entry.Name() == "columns" {
			continue
		}

		// Check if this is a column directory
		columnDir := filepath.Join(boardDir, entry.Name())
		columnMdPath := filepath.Join(columnDir, "column.md")
		metadataYamlPath := filepath.Join(columnDir, "metadata.yml")

		hasColumnMd, _ := filesystem.Exists(columnMdPath)
		hasMetadataYaml, _ := filesystem.Exists(metadataYamlPath)

		if hasColumnMd || hasMetadataYaml {
			columnsToMigrate = append(columnsToMigrate, entry.Name())
		}
	}

	// If no columns to migrate, nothing to do
	if len(columnsToMigrate) == 0 {
		return nil
	}

	// Create columns/ directory
	if err := filesystem.EnsureDir(columnsDir, 0755); err != nil {
		return fmt.Errorf("failed to create columns directory: %w", err)
	}

	// Move each column directory
	for _, columnName := range columnsToMigrate {
		oldPath := filepath.Join(boardDir, columnName)
		newPath := filepath.Join(columnsDir, columnName)

		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move column %s: %w", columnName, err)
		}
	}

	return nil
}

// MigrateColumnsToNewFormat migrates old column format to new normalized format
func (r *BoardRepositoryImpl) MigrateColumnsToNewFormat(ctx context.Context, boardID string) error {
	boardDir, err := r.pathBuilder.BoardDir(boardID)
	if err != nil {
		return err
	}
	columnsDir := filepath.Join(boardDir, "columns")

	// Check if columns/ directory exists
	exists, err := filesystem.Exists(columnsDir)
	if err != nil {
		return fmt.Errorf("failed to check columns directory: %w", err)
	}

	// If columns/ doesn't exist, try legacy location
	if !exists {
		columnsDir = boardDir
	}

	entries, err := os.ReadDir(columnsDir)
	if err != nil {
		return fmt.Errorf("failed to read columns directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip non-column directories
		if entry.Name() == "columns" {
			continue
		}

		columnFolderName := entry.Name()

		// Check if this column already has metadata.yml (already migrated)
		metadataYamlPath, err := r.pathBuilder.ColumnMetadataYaml(boardID, columnFolderName)
		if err != nil {
			return err
		}
		if exists, _ := filesystem.Exists(metadataYamlPath); exists {
			continue
		}

		// Check if this is a column directory (has column.md with frontmatter)
		contentPath, err := r.pathBuilder.ColumnContent(boardID, columnFolderName)
		if err != nil {
			return err
		}
		if exists, _ := filesystem.Exists(contentPath); !exists {
			continue
		}

		// Load the column using old format
		data, err := os.ReadFile(contentPath)
		if err != nil {
			continue
		}

		doc, err := serialization.ParseFrontmatter(data)
		if err != nil {
			// If it's not frontmatter, it might already be in new format
			continue
		}

		// Generate normalized folder name
		normalizedName := slug.Generate(columnFolderName)

		// If normalized name is different, we need to rename the folder
		if normalizedName != columnFolderName {
			oldColumnDir, err := r.pathBuilder.ColumnDir(boardID, columnFolderName)
			if err != nil {
				return err
			}
			newColumnDir, err := r.pathBuilder.ColumnDir(boardID, normalizedName)
			if err != nil {
				return err
			}

			// Rename the folder
			if err := os.Rename(oldColumnDir, newColumnDir); err != nil {
				return fmt.Errorf("failed to rename column folder %s to %s: %w", columnFolderName, normalizedName, err)
			}

			columnFolderName = normalizedName
		}

		// Extract data from old frontmatter format
		displayName := doc.GetString("display_name")
		if displayName == "" {
			displayName = entry.Name()
		}
		description := doc.GetString("description")

		// Create metadata.yml (without display_name and description)
		storage := mapper.ColumnStorage{
			Order:    doc.GetInt("order"),
			WIPLimit: doc.GetInt("wip_limit"),
			Color:    doc.GetString("color"),
		}

		yamlData, err := serialization.SerializeYaml(storage)
		if err != nil {
			return fmt.Errorf("failed to serialize column metadata: %w", err)
		}

		metadataYamlPath, err = r.pathBuilder.ColumnMetadataYaml(boardID, columnFolderName)
		if err != nil {
			return err
		}
		if err := filesystem.SafeWrite(metadataYamlPath, yamlData, 0644); err != nil {
			return fmt.Errorf("failed to write metadata.yml: %w", err)
		}

		// Create new column.md with display name and description
		markdownContent := serialization.SerializeMarkdownWithTitle(displayName, description)
		contentPath, err = r.pathBuilder.ColumnContent(boardID, columnFolderName)
		if err != nil {
			return err
		}
		if err := filesystem.SafeWrite(contentPath, markdownContent, 0644); err != nil {
			return fmt.Errorf("failed to write column.md: %w", err)
		}
	}

	return nil
}

// MigrateTasksToSubdirectory migrates tasks from column root to tasks/ subdirectory
func (r *BoardRepositoryImpl) MigrateTasksToSubdirectory(ctx context.Context, boardID string) error {
	boardDir, err := r.pathBuilder.BoardDir(boardID)
	if err != nil {
		return err
	}
	columnsDir := filepath.Join(boardDir, "columns")

	// Check if columns/ directory exists
	exists, err := filesystem.Exists(columnsDir)
	if err != nil {
		return fmt.Errorf("failed to check columns directory: %w", err)
	}

	// If columns/ doesn't exist, try legacy location (columns directly under board)
	if !exists {
		columnsDir = boardDir
	}

	entries, err := os.ReadDir(columnsDir)
	if err != nil {
		return fmt.Errorf("failed to read columns directory: %w", err)
	}

	// Process each column directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip the columns directory itself if we're in legacy location
		if entry.Name() == "columns" {
			continue
		}

		columnFolderName := entry.Name()
		columnDir := filepath.Join(columnsDir, columnFolderName)
		tasksDir := filepath.Join(columnDir, "tasks")

		// Check if tasks/ subdirectory already exists
		tasksExists, err := filesystem.Exists(tasksDir)
		if err != nil {
			return fmt.Errorf("failed to check tasks directory for column %s: %w", columnFolderName, err)
		}

		// If tasks/ already exists, assume migration is done for this column
		if tasksExists {
			continue
		}

		// Check if this is actually a column directory
		metadataYamlPath := filepath.Join(columnDir, "metadata.yml")
		columnMdPath := filepath.Join(columnDir, "column.md")
		hasMetadata, _ := filesystem.Exists(metadataYamlPath)
		hasColumnMd, _ := filesystem.Exists(columnMdPath)

		if !hasMetadata && !hasColumnMd {
			// Not a column directory
			continue
		}

		// Find all task directories in column root
		columnEntries, err := os.ReadDir(columnDir)
		if err != nil {
			continue
		}

		tasksToMigrate := []string{}
		for _, columnEntry := range columnEntries {
			if !columnEntry.IsDir() {
				continue
			}

			// Check if this is a task directory (has the format PREFIX-NUMBER-slug)
			if r.isTaskDirectory(columnEntry.Name()) {
				tasksToMigrate = append(tasksToMigrate, columnEntry.Name())
			}
		}

		// If no tasks to migrate, skip this column
		if len(tasksToMigrate) == 0 {
			continue
		}

		// Create tasks/ directory
		if err := filesystem.EnsureDir(tasksDir, 0755); err != nil {
			return fmt.Errorf("failed to create tasks directory for column %s: %w", columnFolderName, err)
		}

		// Move each task directory
		for _, taskFolderName := range tasksToMigrate {
			oldPath := filepath.Join(columnDir, taskFolderName)
			newPath := filepath.Join(tasksDir, taskFolderName)

			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to move task %s in column %s: %w", taskFolderName, columnFolderName, err)
			}
		}
	}

	return nil
}
