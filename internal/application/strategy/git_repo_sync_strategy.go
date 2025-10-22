package strategy

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
	"mkanban/pkg/slug"
)

// GitRepoSyncStrategy synchronizes boards for git repository sessions
// Creates tasks for each git branch, with the current branch in "In Progress"
type GitRepoSyncStrategy struct {
	vcsProvider service.VCSProvider
}

// NewGitRepoSyncStrategy creates a new GitRepoSyncStrategy
func NewGitRepoSyncStrategy(vcsProvider service.VCSProvider) *GitRepoSyncStrategy {
	return &GitRepoSyncStrategy{
		vcsProvider: vcsProvider,
	}
}

// CanHandle checks if this session is in a git repository
func (s *GitRepoSyncStrategy) CanHandle(session *entity.Session) bool {
	return s.vcsProvider.IsRepository(session.WorkingDir())
}

// GetBoardName returns the session name as the board name
func (s *GitRepoSyncStrategy) GetBoardName(session *entity.Session) string {
	return session.Name()
}

// Sync synchronizes the board with the git repository state
func (s *GitRepoSyncStrategy) Sync(session *entity.Session, board *entity.Board) error {
	workingDir := session.WorkingDir()

	// Get repository root
	repoRoot, err := s.vcsProvider.GetRepositoryRoot(workingDir)
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get current branch
	currentBranch, err := s.vcsProvider.GetCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// List all branches
	branches, err := s.vcsProvider.ListBranches(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	// Build a set of existing branch tasks for tracking
	existingBranchTasks := make(map[string]*entity.Task)
	for _, column := range board.Columns() {
		for _, task := range column.Tasks() {
			if branchName, exists := task.GetMetadata("git_branch"); exists {
				existingBranchTasks[branchName] = task
			}
		}
	}

	// Create a set of current branches for tracking deletions
	currentBranches := make(map[string]bool)
	for _, branch := range branches {
		currentBranches[branch] = true
	}

	// Process each branch
	for _, branch := range branches {
		isCurrent := branch == currentBranch

		// Check if task already exists for this branch
		if existingTask, exists := existingBranchTasks[branch]; exists {
			// Update existing task
			if err := s.updateBranchTask(board, existingTask, branch, isCurrent, repoRoot); err != nil {
				return fmt.Errorf("failed to update branch task %s: %w", branch, err)
			}
		} else {
			// Create new task for this branch
			if err := s.createBranchTask(board, branch, isCurrent, repoRoot); err != nil {
				return fmt.Errorf("failed to create branch task %s: %w", branch, err)
			}
		}
	}

	// Move deleted branches to "Done" column
	for branchName, task := range existingBranchTasks {
		if !currentBranches[branchName] {
			if err := s.moveTaskToDone(board, task); err != nil {
				// Log error but don't fail the sync
				// We'll retry on next sync
				continue
			}
		}
	}

	return nil
}

// ShouldWatch returns true as git repositories need watching for branch changes
func (s *GitRepoSyncStrategy) ShouldWatch() bool {
	return true
}

// GetWatchPath returns the git refs path for watching branch changes
func (s *GitRepoSyncStrategy) GetWatchPath(session *entity.Session) string {
	repoRoot, err := s.vcsProvider.GetRepositoryRoot(session.WorkingDir())
	if err != nil {
		return ""
	}
	return s.vcsProvider.GetRefsPath(repoRoot)
}

// createBranchTask creates a new task for a git branch
func (s *GitRepoSyncStrategy) createBranchTask(
	board *entity.Board,
	branchName string,
	isCurrent bool,
	repoRoot string,
) error {
	// Determine target column based on whether it's the current branch
	targetColumnName := "To Do"
	if isCurrent {
		targetColumnName = "In Progress"
	}

	// Get or ensure column exists
	column, err := board.GetColumn(targetColumnName)
	if err != nil {
		// Column doesn't exist, we need to create it
		// This shouldn't happen with default boards, but handle it gracefully
		return fmt.Errorf("column %s not found: %w", targetColumnName, err)
	}

	// Check WIP limit
	if !column.CanAddTask() {
		// Column is full, skip creating this task for now
		// It will be created on next sync when there's space
		return nil
	}

	// Generate task ID
	taskSlug := slug.Generate(branchName)
	taskID, err := board.GenerateNextTaskID(taskSlug)
	if err != nil {
		return fmt.Errorf("failed to generate task ID: %w", err)
	}

	// Create task
	task, err := entity.NewTask(
		taskID,
		branchName,
		fmt.Sprintf("Git branch: %s", branchName),
		valueobject.PriorityNone,
		valueobject.StatusTodo,
	)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Set metadata
	task.SetMetadata("git_branch", branchName)
	task.SetMetadata("is_current_branch", fmt.Sprintf("%t", isCurrent))
	task.SetMetadata("repo_path", repoRoot)

	// Add task to column
	if err := column.AddTask(task); err != nil {
		return fmt.Errorf("failed to add task to column: %w", err)
	}

	return nil
}

// updateBranchTask updates an existing branch task's position and metadata
func (s *GitRepoSyncStrategy) updateBranchTask(
	board *entity.Board,
	task *entity.Task,
	branchName string,
	isCurrent bool,
	repoRoot string,
) error {
	// Update metadata
	task.SetMetadata("is_current_branch", fmt.Sprintf("%t", isCurrent))
	task.SetMetadata("repo_path", repoRoot)

	// Determine target column
	targetColumnName := "To Do"
	if isCurrent {
		targetColumnName = "In Progress"
	}

	// Find current column
	_, currentColumn, err := board.FindTask(task.ID())
	if err != nil {
		return fmt.Errorf("task not found in board: %w", err)
	}

	// Move task if needed
	if currentColumn.Name() != targetColumnName {
		// Don't move tasks that are in "Done" column
		// They may have been manually completed
		if currentColumn.Name() != "Done" {
			if err := board.MoveTask(task.ID(), targetColumnName); err != nil {
				// If move fails due to WIP limit, skip for now
				// It will be retried on next sync
				return nil
			}
		}
	}

	return nil
}

// moveTaskToDone moves a task to the "Done" column (for deleted branches)
func (s *GitRepoSyncStrategy) moveTaskToDone(board *entity.Board, task *entity.Task) error {
	_, currentColumn, err := board.FindTask(task.ID())
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// Skip if already in Done
	if currentColumn.Name() == "Done" {
		return nil
	}

	// Move to Done
	if err := board.MoveTask(task.ID(), "Done"); err != nil {
		return fmt.Errorf("failed to move task to Done: %w", err)
	}

	// Mark task as completed
	if err := task.MarkAsCompleted(); err != nil {
		return fmt.Errorf("failed to mark task as completed: %w", err)
	}

	return nil
}
