//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"

	"mkanban/internal/application/usecase/board"
	"mkanban/internal/application/usecase/column"
	"mkanban/internal/application/usecase/task"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/internal/infrastructure/config"
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
