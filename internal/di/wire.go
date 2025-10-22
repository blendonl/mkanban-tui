//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"

	"mkanban/internal/application/strategy"
	"mkanban/internal/application/usecase/board"
	"mkanban/internal/application/usecase/column"
	"mkanban/internal/application/usecase/session"
	"mkanban/internal/application/usecase/task"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/internal/infrastructure/config"
	"mkanban/internal/infrastructure/external"
	"mkanban/internal/infrastructure/persistence/filesystem"
)

// Container holds all application dependencies
type Container struct {
	// Config
	Config *config.Config

	// Repositories
	BoardRepo repository.BoardRepository

	// Domain Services
	ValidationService *service.ValidationService
	BoardService      *service.BoardService
	SessionTracker    service.SessionTracker
	VCSProvider       service.VCSProvider
	ChangeWatcher     service.ChangeWatcher

	// Strategies
	BoardSyncStrategies []strategy.BoardSyncStrategy

	// Use Cases - Board
	CreateBoardUseCase *board.CreateBoardUseCase
	GetBoardUseCase    *board.GetBoardUseCase
	ListBoardsUseCase  *board.ListBoardsUseCase

	// Use Cases - Column
	CreateColumnUseCase *column.CreateColumnUseCase

	// Use Cases - Task
	CreateTaskUseCase *task.CreateTaskUseCase
	MoveTaskUseCase   *task.MoveTaskUseCase
	UpdateTaskUseCase *task.UpdateTaskUseCase
	ListTasksUseCase  *task.ListTasksUseCase

	// Use Cases - Session
	TrackSessionsUseCase        *session.TrackSessionsUseCase
	GetActiveSessionBoardUseCase *session.GetActiveSessionBoardUseCase
	SyncSessionBoardUseCase     *session.SyncSessionBoardUseCase
}

// InitializeContainer sets up all dependencies
func InitializeContainer() (*Container, error) {
	wire.Build(
		// Config
		ProvideConfig,

		// Repositories
		ProvideBoardRepository,

		// Domain Services
		ProvideValidationService,
		ProvideBoardService,
		ProvideSessionTracker,
		ProvideVCSProvider,
		ProvideChangeWatcher,

		// Strategies
		ProvideBoardSyncStrategies,

		// Use Cases - Board
		board.NewCreateBoardUseCase,
		board.NewGetBoardUseCase,
		board.NewListBoardsUseCase,

		// Use Cases - Column
		column.NewCreateColumnUseCase,

		// Use Cases - Task
		task.NewCreateTaskUseCase,
		task.NewMoveTaskUseCase,
		task.NewUpdateTaskUseCase,
		task.NewListTasksUseCase,

		// Use Cases - Session
		session.NewTrackSessionsUseCase,
		session.NewGetActiveSessionBoardUseCase,
		session.NewSyncSessionBoardUseCase,

		// Wire the container
		wire.Struct(new(Container), "*"),
	)
	return nil, nil
}

// Provider functions

func ProvideConfig() (*config.Config, error) {
	loader, err := config.NewLoader()
	if err != nil {
		return nil, err
	}
	return loader.Load()
}

func ProvideBoardRepository(cfg *config.Config) repository.BoardRepository {
	return filesystem.NewBoardRepository(cfg.Storage.BoardsPath)
}

func ProvideValidationService(boardRepo repository.BoardRepository) *service.ValidationService {
	return service.NewValidationService(boardRepo)
}

func ProvideBoardService(
	boardRepo repository.BoardRepository,
	validationService *service.ValidationService,
) *service.BoardService {
	return service.NewBoardService(boardRepo, validationService)
}

func ProvideSessionTracker() service.SessionTracker {
	return external.NewTmuxSessionTracker()
}

func ProvideVCSProvider() service.VCSProvider {
	return external.NewGitVCSProvider()
}

func ProvideChangeWatcher() (service.ChangeWatcher, error) {
	return external.NewFSNotifyWatcher()
}

func ProvideBoardSyncStrategies(
	vcsProvider service.VCSProvider,
	cfg *config.Config,
) []strategy.BoardSyncStrategy {
	strategies := make([]strategy.BoardSyncStrategy, 0)

	// Add GitRepoSyncStrategy (check first, higher priority)
	gitStrategy := strategy.NewGitRepoSyncStrategy(vcsProvider)
	strategies = append(strategies, gitStrategy)

	// Add GeneralSyncStrategy (fallback, lower priority)
	generalStrategy := strategy.NewGeneralSyncStrategy(cfg.SessionTracking.GeneralBoardName)
	strategies = append(strategies, generalStrategy)

	return strategies
}
