package repository

import (
	"context"
	"mkanban/internal/domain/entity"
	"time"
)

type NoteRepository interface {
	Save(ctx context.Context, note *entity.Note) error
	FindByID(ctx context.Context, id string) (*entity.Note, error)
	FindByProject(ctx context.Context, projectID string) ([]*entity.Note, error)
	FindByDate(ctx context.Context, projectID string, date time.Time) ([]*entity.Note, error)
	FindByDateRange(ctx context.Context, projectID string, start, end time.Time) ([]*entity.Note, error)
	FindByType(ctx context.Context, projectID string, noteType entity.NoteType) ([]*entity.Note, error)
	FindByTag(ctx context.Context, projectID string, tag string) ([]*entity.Note, error)
	Search(ctx context.Context, projectID string, query string) ([]*entity.Note, error)
	Delete(ctx context.Context, id string) error
	FindGlobal(ctx context.Context) ([]*entity.Note, error)
	FindGlobalByDate(ctx context.Context, date time.Time) ([]*entity.Note, error)
}
