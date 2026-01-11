package repository

import (
	"context"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"time"
)

type TimeLogRepository interface {
	Save(ctx context.Context, log *entity.TimeLog) error
	FindByID(ctx context.Context, id string) (*entity.TimeLog, error)
	FindByProject(ctx context.Context, projectID string) ([]*entity.TimeLog, error)
	FindByTask(ctx context.Context, taskID *valueobject.TaskID) ([]*entity.TimeLog, error)
	FindByDateRange(ctx context.Context, projectID string, start, end time.Time) ([]*entity.TimeLog, error)
	FindRunning(ctx context.Context) ([]*entity.TimeLog, error)
	Delete(ctx context.Context, id string) error
}
