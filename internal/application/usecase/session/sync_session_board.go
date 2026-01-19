package session

import (
	"context"
	"fmt"
	"mkanban/internal/application/strategy"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
	"mkanban/pkg/slug"
)

// SyncSessionBoardUseCase synchronizes a session's board with its current state
type SyncSessionBoardUseCase struct {
	boardRepo         repository.BoardRepository
	projectRepo       repository.ProjectRepository
	boardService      *service.BoardService
	strategies        []strategy.BoardSyncStrategy
	boardPlanner      *SessionBoardPlanner
}

// NewSyncSessionBoardUseCase creates a new SyncSessionBoardUseCase
func NewSyncSessionBoardUseCase(
	boardRepo repository.BoardRepository,
	projectRepo repository.ProjectRepository,
	boardService *service.BoardService,
	strategies []strategy.BoardSyncStrategy,
	boardPlanner *SessionBoardPlanner,
) *SyncSessionBoardUseCase {
	return &SyncSessionBoardUseCase{
		boardRepo:         boardRepo,
		projectRepo:       projectRepo,
		boardService:      boardService,
		strategies:        strategies,
		boardPlanner:      boardPlanner,
	}
}

// Execute synchronizes the board for the given session
func (uc *SyncSessionBoardUseCase) Execute(ctx context.Context, session *entity.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	plan, err := uc.boardPlanner.Plan(session)
	if err != nil {
		return fmt.Errorf("failed to plan session boards: %w", err)
	}

	if _, err := uc.getOrCreateProject(ctx, plan, session); err != nil {
		return fmt.Errorf("failed to get or create project: %w", err)
	}

	var selectedStrategy strategy.BoardSyncStrategy
	for _, strat := range uc.strategies {
		if strat.CanHandle(session) {
			selectedStrategy = strat
			break
		}
	}

	if selectedStrategy == nil {
		return fmt.Errorf("no strategy found for session: %s", session.Name())
	}

	for _, boardName := range plan.BoardNames {
		board, err := uc.getOrCreateBoard(ctx, plan.ProjectID, boardName, session)
		if err != nil {
			return fmt.Errorf("failed to get or create board: %w", err)
		}

		if boardName == plan.SyncBoardName {
			if err := selectedStrategy.Sync(session, board); err != nil {
				return fmt.Errorf("failed to sync board: %w", err)
			}
		}

		if err := uc.boardRepo.Save(ctx, board); err != nil {
			return fmt.Errorf("failed to save board: %w", err)
		}
	}

	return nil
}

// getOrCreateBoard retrieves an existing board or creates a new one
func (uc *SyncSessionBoardUseCase) getOrCreateBoard(
	ctx context.Context,
	projectID string,
	boardName string,
	session *entity.Session,
) (*entity.Board, error) {
	boardSlug := slug.Generate(boardName)
	boardID, err := valueobject.BuildBoardID(projectID, boardSlug)
	if err != nil {
		return nil, err
	}

	board, err := uc.boardRepo.FindByID(ctx, boardID)
	if err == nil {
		return board, nil
	}

	if err != entity.ErrBoardNotFound {
		return nil, fmt.Errorf("failed to check for existing board: %w", err)
	}

	description := fmt.Sprintf("Session: %s\nWorking Directory: %s",
		session.Name(), session.WorkingDir())

	board, err = uc.boardService.CreateBoard(ctx, projectID, boardName, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create board: %w", err)
	}

	if err := uc.addDefaultColumns(board); err != nil {
		return nil, fmt.Errorf("failed to add default columns: %w", err)
	}

	if err := uc.boardRepo.Save(ctx, board); err != nil {
		return nil, fmt.Errorf("failed to save board with columns: %w", err)
	}

	return board, nil
}

// addDefaultColumns adds the default kanban columns to a new board
func (uc *SyncSessionBoardUseCase) addDefaultColumns(board *entity.Board) error {
	// Default columns for a kanban board
	defaultColumns := []struct {
		name        string
		description string
		order       int
		wipLimit    int
	}{
		{"To Do", "Tasks to be started", 0, 0},
		{"In Progress", "Tasks currently being worked on", 1, 3},
		{"Done", "Completed tasks", 2, 0},
	}

	for _, col := range defaultColumns {
		column, err := entity.NewColumn(col.name, col.description, col.order, col.wipLimit, nil)
		if err != nil {
			return fmt.Errorf("failed to create column %s: %w", col.name, err)
		}

		if err := board.AddColumn(column); err != nil {
			return fmt.Errorf("failed to add column %s to board: %w", col.name, err)
		}
	}

	return nil
}

func (uc *SyncSessionBoardUseCase) getOrCreateProject(
	ctx context.Context,
	plan *SessionBoardPlan,
	session *entity.Session,
) (*entity.Project, error) {
	project, err := uc.projectRepo.FindBySlug(ctx, plan.ProjectID)
	if err == nil && project != nil {
		if plan.WorkingDir != "" && project.WorkingDir() != plan.WorkingDir {
			project.SetWorkingDir(plan.WorkingDir)
			if err := uc.projectRepo.Save(ctx, project); err != nil {
				return nil, fmt.Errorf("failed to update project: %w", err)
			}
		}
		return project, nil
	}
	if err != nil && err != entity.ErrProjectNotFound {
		return nil, err
	}

	description := fmt.Sprintf("Session: %s\nWorking Directory: %s",
		session.Name(), plan.WorkingDir)
	project, err = entity.NewProject(plan.ProjectID, plan.ProjectName, description)
	if err != nil {
		return nil, err
	}
	if plan.WorkingDir != "" {
		project.SetWorkingDir(plan.WorkingDir)
	}
	if err := uc.projectRepo.Save(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return project, nil
}
