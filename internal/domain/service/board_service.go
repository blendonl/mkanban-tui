package service

import (
	"context"
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/config"
	"mkanban/pkg/slug"
)

// BoardService provides high-level domain operations for boards
type BoardService struct {
	boardRepo         repository.BoardRepository
	validationService *ValidationService
	config            *config.Config
}

// NewBoardService creates a new BoardService
func NewBoardService(
	boardRepo repository.BoardRepository,
	validationService *ValidationService,
	cfg *config.Config,
) *BoardService {
	return &BoardService{
		boardRepo:         boardRepo,
		validationService: validationService,
		config:            cfg,
	}
}

// CreateBoard creates a new board with validation
func (s *BoardService) CreateBoard(ctx context.Context, projectID, name, description string) (*entity.Board, error) {
	// Validate board name
	if err := s.validationService.ValidateBoardName(ctx, name); err != nil {
		return nil, err
	}

	// Check uniqueness
	if err := s.validationService.ValidateUniqueBoardName(ctx, projectID, name, ""); err != nil {
		return nil, err
	}

	// Generate board ID from project and board name
	boardSlug := slug.Generate(name)
	boardID, err := valueobject.BuildBoardID(projectID, boardSlug)
	if err != nil {
		return nil, err
	}

	// Check if board with this ID already exists
	exists, err := s.boardRepo.Exists(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to check board existence: %w", err)
	}
	if exists {
		return nil, entity.ErrBoardAlreadyExists
	}

	// Create the board
	board, err := entity.NewBoard(boardID, name, description)
	if err != nil {
		return nil, err
	}
	board.SetProjectID(projectID)

	// Persist to repository
	if err := s.boardRepo.Save(ctx, board); err != nil {
		return nil, fmt.Errorf("failed to save board: %w", err)
	}

	return board, nil
}

// AddColumnToBoard adds a new column to a board
func (s *BoardService) AddColumnToBoard(
	ctx context.Context,
	boardID string,
	columnName string,
	description string,
	order int,
	wipLimit int,
	color *valueobject.Color,
) (*entity.Board, error) {
	// Load board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, err
	}

	// Validate column name
	if err := s.validationService.ValidateColumnName(columnName); err != nil {
		return nil, err
	}

	// Check uniqueness within board (check display names)
	if err := s.validationService.ValidateUniqueColumnName(board, columnName, nil); err != nil {
		return nil, err
	}

	// Validate WIP limit
	if err := s.validationService.ValidateWIPLimit(wipLimit); err != nil {
		return nil, err
	}

	// Generate normalized folder name from display name
	normalizedName := slug.Generate(columnName)

	// Create column with both normalized and display names
	column, err := entity.NewColumnWithDisplayName(normalizedName, columnName, description, order, wipLimit, color)
	if err != nil {
		return nil, err
	}

	// Add to board
	if err := board.AddColumn(column); err != nil {
		return nil, err
	}

	// Reorder columns
	board.ReorderColumns()

	// Save board
	if err := s.boardRepo.Save(ctx, board); err != nil {
		return nil, fmt.Errorf("failed to save board: %w", err)
	}

	return board, nil
}

// CreateTask creates a new task in a specific column
func (s *BoardService) CreateTask(
	ctx context.Context,
	boardID string,
	columnName string,
	title string,
	description string,
	priority valueobject.Priority,
) (*entity.Board, *entity.Task, error) {
	// Load board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, nil, err
	}

	// Validate task title
	if err := s.validationService.ValidateTaskTitle(title); err != nil {
		return nil, nil, err
	}

	// Get column
	column, err := board.GetColumn(columnName)
	if err != nil {
		return nil, nil, err
	}

	// Check WIP limit
	if !column.CanAddTask() {
		return nil, nil, entity.ErrWIPLimitExceeded
	}

	// Generate task ID
	taskSlug := slug.Generate(title)
	taskID, err := board.GenerateNextTaskID(taskSlug)
	if err != nil {
		return nil, nil, err
	}

	// Create task
	task, err := entity.NewTask(taskID, title, description, priority, valueobject.StatusTodo)
	if err != nil {
		return nil, nil, err
	}

	// Add task to column
	if err := column.AddTask(task); err != nil {
		return nil, nil, err
	}

	// Parse description for subtasks
	subtaskTitles := ParseSubtasks(description)
	if len(subtaskTitles) > 0 {
		// Get the first column (Todo) for subtasks
		var todoColumn *entity.Column
		if board.ColumnCount() > 0 {
			todoColumn, _ = board.GetColumnByIndex(0)
		}

		if todoColumn != nil {
			updatedDescription := description

			// Create a subtask for each checkbox
			for _, subtaskTitle := range subtaskTitles {
				// Generate subtask ID
				subtaskSlug := slug.Generate(subtaskTitle)
				subtaskID, err := board.GenerateNextTaskID(subtaskSlug)
				if err != nil {
					continue // Skip this subtask on error
				}

				// Create subtask
				subtask, err := entity.NewTask(subtaskID, subtaskTitle, "", priority, valueobject.StatusTodo)
				if err != nil {
					continue // Skip this subtask on error
				}

				// Set parent ID
				subtask.SetParentID(taskID)

				// Add subtask to todo column
				if err := todoColumn.AddTask(subtask); err != nil {
					continue // Skip this subtask on error
				}

				// Update parent description with markdown link
				updatedDescription = AddSubtaskLink(updatedDescription, subtaskTitle, subtaskID.String(), todoColumn.Name())
			}

			// Update parent task description with links
			task.UpdateDescription(updatedDescription)
		}
	}

	// Save board
	if err := s.boardRepo.Save(ctx, board); err != nil {
		return nil, nil, fmt.Errorf("failed to save board: %w", err)
	}

	return board, task, nil
}

// MoveTask moves a task between columns
func (s *BoardService) MoveTask(
	ctx context.Context,
	boardID string,
	taskID *valueobject.TaskID,
	targetColumnName string,
) (*entity.Board, error) {
	// Load board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, err
	}

	// Find the task being moved
	task, _, err := board.FindTask(taskID)
	if err != nil {
		return nil, err
	}

	// Move task
	if err := board.MoveTask(taskID, targetColumnName); err != nil {
		return nil, err
	}

	// If this task has a parent, update the parent's checkbox state
	if task.IsSubtask() {
		parentTask, _, err := board.FindTask(task.ParentID())
		if err == nil {
			// Determine checkbox state based on target column
			var checkboxState CheckboxState
			switch targetColumnName {
			case "Done":
				checkboxState = CheckboxDone
			case "In Progress":
				checkboxState = CheckboxInProgress
			default:
				checkboxState = CheckboxTodo
			}

			// Update the parent's description
			updatedDescription := UpdateCheckboxState(
				parentTask.Description(),
				taskID.String(),
				checkboxState,
			)
			parentTask.UpdateDescription(updatedDescription)

			// Check if all subtasks are complete
			if AllCheckboxesComplete(parentTask.Description()) {
				// Move parent to Done column
				doneColumn, err := board.GetColumn("Done")
				if err == nil && doneColumn.CanAddTask() {
					_ = board.MoveTask(task.ParentID(), "Done")
				}
			}
		}
	}

	// Save board
	if err := s.boardRepo.Save(ctx, board); err != nil {
		return nil, fmt.Errorf("failed to save board: %w", err)
	}

	return board, nil
}

// DeleteTask deletes a task from the board
func (s *BoardService) DeleteTask(
	ctx context.Context,
	boardID string,
	taskID *valueobject.TaskID,
) (*entity.Board, error) {
	// Load board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, err
	}

	// Find task and its column
	_, column, err := board.FindTask(taskID)
	if err != nil {
		return nil, err
	}

	// Remove task from column
	if _, err := column.RemoveTask(taskID); err != nil {
		return nil, err
	}

	// Save board
	if err := s.boardRepo.Save(ctx, board); err != nil {
		return nil, fmt.Errorf("failed to save board: %w", err)
	}

	return board, nil
}

// CheckAndCompleteParent checks if all subtasks of a parent are complete
// and moves the parent to Done if so
func (s *BoardService) CheckAndCompleteParent(
	ctx context.Context,
	board *entity.Board,
	parentTaskID *valueobject.TaskID,
) error {
	parentTask, _, err := board.FindTask(parentTaskID)
	if err != nil {
		return err
	}

	// Check if all subtasks are complete
	if AllCheckboxesComplete(parentTask.Description()) {
		// Move parent to Done column
		doneColumn, err := board.GetColumn("Done")
		if err == nil && doneColumn.CanAddTask() {
			return board.MoveTask(parentTaskID, "Done")
		}
	}

	return nil
}

// UpdateTask updates task details
func (s *BoardService) UpdateTask(
	ctx context.Context,
	boardID string,
	taskID *valueobject.TaskID,
	title *string,
	description *string,
	priority *valueobject.Priority,
	status *valueobject.Status,
) (*entity.Board, *entity.Task, error) {
	// Load board
	board, err := s.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return nil, nil, err
	}

	// Find task
	task, _, err := board.FindTask(taskID)
	if err != nil {
		return nil, nil, err
	}

	// Update fields if provided
	if title != nil {
		if err := s.validationService.ValidateTaskTitle(*title); err != nil {
			return nil, nil, err
		}
		if err := task.UpdateTitle(*title); err != nil {
			return nil, nil, err
		}
	}

	if description != nil {
		task.UpdateDescription(*description)
	}

	if priority != nil {
		if err := task.UpdatePriority(*priority); err != nil {
			return nil, nil, err
		}
	}

	if status != nil {
		if err := task.UpdateStatus(*status); err != nil {
			return nil, nil, err
		}
	}

	// Save board
	if err := s.boardRepo.Save(ctx, board); err != nil {
		return nil, nil, fmt.Errorf("failed to save board: %w", err)
	}

	return board, task, nil
}
