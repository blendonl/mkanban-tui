//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"

	"mkanban/internal/application/strategy"
	"mkanban/internal/application/usecase/action"
	"mkanban/internal/application/usecase/board"
	"mkanban/internal/application/usecase/column"
	"mkanban/internal/application/usecase/session"
	"mkanban/internal/application/usecase/task"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/internal/infrastructure/config"
	"mkanban/internal/infrastructure/external"
	"mkanban/internal/infrastructure/persistence/filesystem"
	infraService "mkanban/internal/infrastructure/service"
)

// Container holds all application dependencies
type Container struct {
	// Config
	Config *config.Config

	// Repositories
	BoardRepo   repository.BoardRepository
	ActionRepo  repository.ActionRepository
	ProjectRepo repository.ProjectRepository
	TimeLogRepo repository.TimeLogRepository
	NoteRepo    repository.NoteRepository

	// Domain Services
	ValidationService *service.ValidationService
	BoardService      *service.BoardService
	SessionTracker    service.SessionTracker
	VCSProvider       service.VCSProvider
	ChangeWatcher     service.ChangeWatcher
	RepoPathResolver  service.RepoPathResolver

	// Strategies
	BoardSyncStrategies []strategy.BoardSyncStrategy

	// Use Cases - Board
	CreateBoardUseCase *board.CreateBoardUseCase
	GetBoardUseCase    *board.GetBoardUseCase
	ListBoardsUseCase  *board.ListBoardsUseCase

	// Use Cases - Column
	CreateColumnUseCase *column.CreateColumnUseCase

	// Use Cases - Task
	CreateTaskUseCase   *task.CreateTaskUseCase
	MoveTaskUseCase     *task.MoveTaskUseCase
	UpdateTaskUseCase   *task.UpdateTaskUseCase
	ListTasksUseCase    *task.ListTasksUseCase
	CheckoutTaskUseCase *task.CheckoutTaskUseCase

	// Use Cases - Session
	TrackSessionsUseCase        *session.TrackSessionsUseCase
	GetActiveSessionBoardUseCase *session.GetActiveSessionBoardUseCase
	SyncSessionBoardUseCase     *session.SyncSessionBoardUseCase

	// Use Cases - Action
	CreateActionUseCase   *action.CreateActionUseCase
	UpdateActionUseCase   *action.UpdateActionUseCase
	DeleteActionUseCase   *action.DeleteActionUseCase
	GetActionUseCase      *action.GetActionUseCase
	ListActionsUseCase    *action.ListActionsUseCase
	EnableActionUseCase   *action.EnableActionUseCase
	DisableActionUseCase  *action.DisableActionUseCase
	EvaluateActionsUseCase *action.EvaluateActionsUseCase
	ExecuteActionUseCase  *action.ExecuteActionUseCase
	ProcessEventUseCase   *action.ProcessEventUseCase

	// Infrastructure Services
	EventBus      entity.EventBus
	Notifier      entity.Notifier
	ScriptRunner  entity.ScriptRunner
	TaskMutator   entity.TaskMutator
}

// InitializeContainer sets up all dependencies
func InitializeContainer() (*Container, error) {
	wire.Build(
		// Config
		ProvideConfig,

		// Repositories
		ProvideBoardRepository,
		ProvideActionRepository,
		ProvideProjectRepository,
		ProvideTimeLogRepository,
		ProvideNoteRepository,

		// Domain Services
		ProvideValidationService,
		ProvideBoardService,
		ProvideSessionTracker,
		ProvideVCSProvider,
		ProvideChangeWatcher,
		ProvideRepoPathResolver,

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
		task.NewCheckoutTaskUseCase,

		// Use Cases - Session
		session.NewSessionBoardPlanner,
		session.NewTrackSessionsUseCase,
		session.NewGetActiveSessionBoardUseCase,
		session.NewSyncSessionBoardUseCase,

		// Infrastructure Services
		ProvideEventBus,
		ProvideNotifier,
		ProvideScriptRunner,
		ProvideTaskMutator,

		// Use Cases - Action
		action.NewCreateActionUseCase,
		action.NewUpdateActionUseCase,
		action.NewDeleteActionUseCase,
		action.NewGetActionUseCase,
		action.NewListActionsUseCase,
		action.NewEnableActionUseCase,
		action.NewDisableActionUseCase,
		action.NewEvaluateActionsUseCase,
		action.NewExecuteActionUseCase,
		action.NewProcessEventUseCase,

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
	return filesystem.NewBoardRepository(cfg.Storage.DataPath)
}

func ProvideValidationService(boardRepo repository.BoardRepository) *service.ValidationService {
	return service.NewValidationService(boardRepo)
}

func ProvideBoardService(
	boardRepo repository.BoardRepository,
	validationService *service.ValidationService,
	cfg *config.Config,
) *service.BoardService {
	return service.NewBoardService(boardRepo, validationService, cfg)
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

func ProvideRepoPathResolver(
	sessionTracker service.SessionTracker,
	vcsProvider service.VCSProvider,
	projectRepo repository.ProjectRepository,
) service.RepoPathResolver {
	return infraService.NewTmuxRepoPathResolver(sessionTracker, vcsProvider, projectRepo)
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

func ProvideActionRepository(cfg *config.Config) repository.ActionRepository {
	return filesystem.NewActionRepository(cfg)
}

func ProvideEventBus() entity.EventBus {
	return infraService.NewEventBus()
}

func ProvideNotifier(cfg *config.Config) entity.Notifier {
	return external.NewDesktopNotifier(cfg.Actions.NotificationsEnabled)
}

func ProvideScriptRunner(cfg *config.Config) entity.ScriptRunner {
	return external.NewScriptExecutor(cfg.Actions.ScriptsEnabled, cfg.Actions.ScriptsDir)
}

func ProvideTaskMutator(
	createTaskUseCase *task.CreateTaskUseCase,
	updateTaskUseCase *task.UpdateTaskUseCase,
	moveTaskUseCase *task.MoveTaskUseCase,
) entity.TaskMutator {
	return infraService.NewTaskMutatorService(createTaskUseCase, updateTaskUseCase, moveTaskUseCase)
}

func ProvideProjectRepository(cfg *config.Config) repository.ProjectRepository {
	return filesystem.NewProjectRepository(cfg.Storage.DataPath)
}

func ProvideTimeLogRepository(cfg *config.Config) repository.TimeLogRepository {
	return filesystem.NewTimeLogRepository(cfg.Storage.DataPath)
}

func ProvideNoteRepository(cfg *config.Config) repository.NoteRepository {
	return filesystem.NewNoteRepository(cfg.Storage.DataPath)
}
