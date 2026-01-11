package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"time"
)

type TimeLogStorage struct {
	ID          string            `yaml:"id"`
	ProjectID   string            `yaml:"project_id"`
	TaskID      string            `yaml:"task_id,omitempty"`
	Source      string            `yaml:"source"`
	StartTime   time.Time         `yaml:"start_time"`
	EndTime     *time.Time        `yaml:"end_time,omitempty"`
	Duration    int64             `yaml:"duration_seconds,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	Created     time.Time         `yaml:"created"`
	Modified    time.Time         `yaml:"modified"`
}

type MonthlyTimeLogStorage struct {
	Logs []*TimeLogStorage `yaml:"logs"`
}

func TimeLogToStorage(log *entity.TimeLog) *TimeLogStorage {
	storage := &TimeLogStorage{
		ID:          log.ID(),
		ProjectID:   log.ProjectID(),
		Source:      log.Source().String(),
		StartTime:   log.StartTime(),
		Description: log.Description(),
		Created:     log.CreatedAt(),
		Modified:    log.ModifiedAt(),
	}

	if log.TaskID() != nil {
		storage.TaskID = log.TaskID().String()
	}

	if log.EndTime() != nil {
		storage.EndTime = log.EndTime()
	}

	if log.Duration() > 0 {
		storage.Duration = int64(log.Duration().Seconds())
	}

	if len(log.Metadata()) > 0 {
		storage.Metadata = log.Metadata()
	}

	return storage
}

func TimeLogFromStorage(storage *TimeLogStorage) (*entity.TimeLog, error) {
	if storage.ID == "" {
		return nil, fmt.Errorf("missing time log ID")
	}

	source := entity.TimeLogSource(storage.Source)
	if !source.IsValid() {
		return nil, fmt.Errorf("invalid time log source: %s", storage.Source)
	}

	log, err := entity.NewTimeLog(storage.ID, storage.ProjectID, source, storage.StartTime)
	if err != nil {
		return nil, err
	}

	if storage.TaskID != "" {
		taskID, err := valueobject.ParseTaskID(storage.TaskID)
		if err == nil {
			log.SetTaskID(taskID)
		}
	}

	if storage.Description != "" {
		log.SetDescription(storage.Description)
	}

	if storage.EndTime != nil {
		_ = log.Stop(*storage.EndTime)
	} else if storage.Duration > 0 {
		_ = log.SetDuration(time.Duration(storage.Duration) * time.Second)
	}

	for key, value := range storage.Metadata {
		log.SetMetadata(key, value)
	}

	return log, nil
}
