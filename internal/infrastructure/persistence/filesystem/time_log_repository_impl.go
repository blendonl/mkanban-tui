package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/persistence/mapper"
	"mkanban/internal/infrastructure/serialization"
	"mkanban/pkg/filesystem"
)

type TimeLogRepositoryImpl struct {
	pathBuilder *ProjectPathBuilder
}

func NewTimeLogRepository(rootPath string) repository.TimeLogRepository {
	return &TimeLogRepositoryImpl{
		pathBuilder: NewProjectPathBuilder(rootPath),
	}
}

func (r *TimeLogRepositoryImpl) Save(ctx context.Context, log *entity.TimeLog) error {
	projectSlug, err := r.getProjectSlug(ctx, log.ProjectID())
	if err != nil {
		return err
	}

	logsDir := r.pathBuilder.ProjectTimeLogsDir(projectSlug)
	if err := filesystem.EnsureDir(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create time logs directory: %w", err)
	}

	yearMonth := log.StartTime().Format("2006-01")
	logFile := r.pathBuilder.TimeLogFile(projectSlug, yearMonth)

	logs, err := r.loadLogsFromFile(logFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing logs: %w", err)
	}

	updated := false
	for i, existing := range logs {
		if existing.ID() == log.ID() {
			logs[i] = log
			updated = true
			break
		}
	}
	if !updated {
		logs = append(logs, log)
	}

	return r.saveLogsToFile(logFile, logs)
}

func (r *TimeLogRepositoryImpl) FindByID(ctx context.Context, id string) (*entity.TimeLog, error) {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, entity.ErrTimeLogNotFound
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		logsDir := r.pathBuilder.ProjectTimeLogsDir(entry.Name())
		logs, err := r.loadAllLogsFromDir(logsDir)
		if err != nil {
			continue
		}

		for _, log := range logs {
			if log.ID() == id {
				return log, nil
			}
		}
	}

	return nil, entity.ErrTimeLogNotFound
}

func (r *TimeLogRepositoryImpl) FindByProject(ctx context.Context, projectID string) ([]*entity.TimeLog, error) {
	projectSlug, err := r.getProjectSlug(ctx, projectID)
	if err != nil {
		return nil, err
	}

	logsDir := r.pathBuilder.ProjectTimeLogsDir(projectSlug)
	return r.loadAllLogsFromDir(logsDir)
}

func (r *TimeLogRepositoryImpl) FindByTask(ctx context.Context, taskID *valueobject.TaskID) ([]*entity.TimeLog, error) {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []*entity.TimeLog{}, nil
		}
		return nil, err
	}

	var result []*entity.TimeLog
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		logsDir := r.pathBuilder.ProjectTimeLogsDir(entry.Name())
		logs, err := r.loadAllLogsFromDir(logsDir)
		if err != nil {
			continue
		}

		for _, log := range logs {
			if log.TaskID() != nil && log.TaskID().Equal(taskID) {
				result = append(result, log)
			}
		}
	}

	return result, nil
}

func (r *TimeLogRepositoryImpl) FindByDateRange(ctx context.Context, projectID string, start, end time.Time) ([]*entity.TimeLog, error) {
	projectSlug, err := r.getProjectSlug(ctx, projectID)
	if err != nil {
		return nil, err
	}

	logsDir := r.pathBuilder.ProjectTimeLogsDir(projectSlug)
	allLogs, err := r.loadAllLogsFromDir(logsDir)
	if err != nil {
		return nil, err
	}

	var result []*entity.TimeLog
	for _, log := range allLogs {
		if (log.StartTime().Equal(start) || log.StartTime().After(start)) &&
			(log.StartTime().Equal(end) || log.StartTime().Before(end)) {
			result = append(result, log)
		}
	}

	return result, nil
}

func (r *TimeLogRepositoryImpl) FindRunning(ctx context.Context) ([]*entity.TimeLog, error) {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []*entity.TimeLog{}, nil
		}
		return nil, err
	}

	var result []*entity.TimeLog
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		logsDir := r.pathBuilder.ProjectTimeLogsDir(entry.Name())
		logs, err := r.loadAllLogsFromDir(logsDir)
		if err != nil {
			continue
		}

		for _, log := range logs {
			if log.IsRunning() {
				result = append(result, log)
			}
		}
	}

	return result, nil
}

func (r *TimeLogRepositoryImpl) Delete(ctx context.Context, id string) error {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		logsDir := r.pathBuilder.ProjectTimeLogsDir(entry.Name())
		files, err := os.ReadDir(logsDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yml") {
				continue
			}

			logFile := filepath.Join(logsDir, file.Name())
			logs, err := r.loadLogsFromFile(logFile)
			if err != nil {
				continue
			}

			for i, log := range logs {
				if log.ID() == id {
					logs = append(logs[:i], logs[i+1:]...)
					return r.saveLogsToFile(logFile, logs)
				}
			}
		}
	}

	return entity.ErrTimeLogNotFound
}

func (r *TimeLogRepositoryImpl) getProjectSlug(ctx context.Context, projectID string) (string, error) {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", entity.ErrProjectNotFound
		}
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadataPath := r.pathBuilder.ProjectMetadata(entry.Name())
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			continue
		}

		var storage mapper.ProjectStorage
		if err := serialization.ParseYaml(data, &storage); err != nil {
			continue
		}

		if storage.ID == projectID {
			return entry.Name(), nil
		}
	}

	return "", entity.ErrProjectNotFound
}

func (r *TimeLogRepositoryImpl) loadLogsFromFile(filePath string) ([]*entity.TimeLog, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var storage mapper.MonthlyTimeLogStorage
	if err := serialization.ParseYaml(data, &storage); err != nil {
		return nil, err
	}

	var logs []*entity.TimeLog
	for _, s := range storage.Logs {
		log, err := mapper.TimeLogFromStorage(s)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func (r *TimeLogRepositoryImpl) loadAllLogsFromDir(logsDir string) ([]*entity.TimeLog, error) {
	exists, err := filesystem.Exists(logsDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []*entity.TimeLog{}, nil
	}

	files, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, err
	}

	var allLogs []*entity.TimeLog
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		logFile := filepath.Join(logsDir, file.Name())
		logs, err := r.loadLogsFromFile(logFile)
		if err != nil {
			continue
		}

		allLogs = append(allLogs, logs...)
	}

	return allLogs, nil
}

func (r *TimeLogRepositoryImpl) saveLogsToFile(filePath string, logs []*entity.TimeLog) error {
	storage := mapper.MonthlyTimeLogStorage{
		Logs: make([]*mapper.TimeLogStorage, 0, len(logs)),
	}

	for _, log := range logs {
		storage.Logs = append(storage.Logs, mapper.TimeLogToStorage(log))
	}

	data, err := serialization.SerializeYaml(storage)
	if err != nil {
		return fmt.Errorf("failed to serialize time logs: %w", err)
	}

	return filesystem.SafeWrite(filePath, data, 0644)
}
