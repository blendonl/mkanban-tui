package task

import (
	"context"
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
	"regexp"
	"strconv"
	"strings"
)

// CheckoutTaskUseCase handles checking out a task (moves to In Progress and checks out git branch)
type CheckoutTaskUseCase struct {
	boardRepo   repository.BoardRepository
	vcsProvider service.VCSProvider
}

// NewCheckoutTaskUseCase creates a new CheckoutTaskUseCase
func NewCheckoutTaskUseCase(
	boardRepo repository.BoardRepository,
	vcsProvider service.VCSProvider,
) *CheckoutTaskUseCase {
	return &CheckoutTaskUseCase{
		boardRepo:   boardRepo,
		vcsProvider: vcsProvider,
	}
}

// Execute checks out a task by ID
func (uc *CheckoutTaskUseCase) Execute(
	ctx context.Context,
	boardID string,
	taskIDStr string,
	branchFormat string,
) error {
	// Load board
	board, err := uc.boardRepo.FindByID(ctx, boardID)
	if err != nil {
		return fmt.Errorf("failed to load board: %w", err)
	}

	// Find task by ID (supports short ID like "REC-007" or full ID)
	task, currentColumn, err := uc.findTaskByID(board, taskIDStr)
	if err != nil {
		return fmt.Errorf("failed to find task: %w", err)
	}

	// Check if we're in a git repository
	workingDir := "."
	if !uc.vcsProvider.IsRepository(workingDir) {
		return fmt.Errorf("not in a git repository")
	}

	// Get repository root
	repoRoot, err := uc.vcsProvider.GetRepositoryRoot(workingDir)
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	// Determine branch name
	var branchName string
	if gitBranch, hasBranch := task.GetMetadata("git_branch"); hasBranch {
		// Task already has a branch, use it
		branchName = gitBranch
	} else {
		// Create new branch name using format
		branchName = FormatBranchName(branchFormat, task.ID(), task.Title())

		// Set metadata for future reference
		task.SetMetadata("git_branch", branchName)
		task.SetMetadata("repo_path", repoRoot)
	}

	// Check if branch exists
	branchExists, err := uc.vcsProvider.BranchExists(repoRoot, branchName)
	if err != nil {
		return fmt.Errorf("failed to check if branch exists: %w", err)
	}

	// Checkout or create branch
	if branchExists {
		if err := uc.vcsProvider.CheckoutBranch(repoRoot, branchName); err != nil {
			return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
		}
		fmt.Printf("Checked out existing branch: %s\n", branchName)
	} else {
		if err := uc.vcsProvider.CreateAndCheckoutBranch(repoRoot, branchName); err != nil {
			return fmt.Errorf("failed to create and checkout branch %s: %w", branchName, err)
		}
		fmt.Printf("Created and checked out new branch: %s\n", branchName)
	}

	// Get columns
	inProgressColumn, err := board.GetColumn("In Progress")
	if err != nil {
		return fmt.Errorf("failed to get In Progress column: %w", err)
	}

	todoColumn, err := board.GetColumn("To Do")
	if err != nil {
		// Try alternative names
		todoColumn, err = board.GetColumn("Todo")
		if err != nil {
			return fmt.Errorf("failed to get To Do column: %w", err)
		}
	}

	// Step 1: Move all in-progress tasks to todo (except the target task if it's already in progress)
	tasksToMove := make([]*entity.Task, 0)
	for _, inProgressTask := range inProgressColumn.Tasks() {
		if !inProgressTask.ID().Equal(task.ID()) {
			tasksToMove = append(tasksToMove, inProgressTask)
		}
	}

	// Actually move the tasks
	for _, taskToMove := range tasksToMove {
		// Use board.MoveTask for proper handling
		if todoColumn.CanAddTask() {
			if err := board.MoveTask(taskToMove.ID(), todoColumn.Name()); err != nil {
				// If can't move, just skip
				continue
			}
		}
	}

	// Step 2: Move target task to "In Progress" if not already there
	// Re-fetch the task's current column after moving other tasks
	_, currentColumn, err = board.FindTask(task.ID())
	if err != nil {
		return fmt.Errorf("failed to find task after moving others: %w", err)
	}

	if currentColumn.Name() != "In Progress" {
		if err := board.MoveTask(task.ID(), "In Progress"); err != nil {
			return fmt.Errorf("failed to move task to In Progress: %w", err)
		}
	}

	// Update task metadata
	task.SetMetadata("is_current_branch", "true")

	// Save board
	if err := uc.boardRepo.Save(ctx, board); err != nil {
		return fmt.Errorf("failed to save board: %w", err)
	}

	fmt.Printf("Task %s (%s) is now in progress\n", task.ID().ShortID(), task.Title())

	return nil
}

// findTaskByID finds a task by full ID or short ID (e.g., "REC-007")
func (uc *CheckoutTaskUseCase) findTaskByID(board *entity.Board, taskIDStr string) (*entity.Task, *entity.Column, error) {
	// First, try to parse as full ID
	fullID, err := valueobject.ParseTaskID(taskIDStr)
	if err == nil {
		// Valid full ID, try to find it
		task, column, err := board.FindTask(fullID)
		if err == nil {
			return task, column, nil
		}
	}

	// Try as short ID (PREFIX-NUMBER)
	shortIDRegex := regexp.MustCompile(`^([A-Z]{3})-(\d+)$`)
	matches := shortIDRegex.FindStringSubmatch(strings.TrimSpace(taskIDStr))

	if matches != nil {
		prefix := matches[1]
		number, _ := strconv.Atoi(matches[2])

		// Search through all tasks to find matching prefix and number
		for _, column := range board.Columns() {
			for _, task := range column.Tasks() {
				if task.ID().Prefix() == prefix && task.ID().Number() == number {
					return task, column, nil
				}
			}
		}
	}

	return nil, nil, fmt.Errorf("task not found: %s", taskIDStr)
}
