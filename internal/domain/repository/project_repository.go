package repository

import (
	"context"
	"mkanban/internal/domain/entity"
)

type ProjectRepository interface {
	Save(ctx context.Context, project *entity.Project) error
	FindByID(ctx context.Context, id string) (*entity.Project, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Project, error)
	FindAll(ctx context.Context) ([]*entity.Project, error)
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
}
