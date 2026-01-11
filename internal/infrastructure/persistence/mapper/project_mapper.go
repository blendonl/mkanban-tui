package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"time"
)

type ProjectStorage struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Slug        string            `yaml:"slug"`
	Description string            `yaml:"description,omitempty"`
	Color       string            `yaml:"color,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Archived    bool              `yaml:"archived,omitempty"`
	Created     time.Time         `yaml:"created"`
	Modified    time.Time         `yaml:"modified"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

func ProjectToStorage(project *entity.Project) *ProjectStorage {
	storage := &ProjectStorage{
		ID:          project.ID(),
		Name:        project.Name(),
		Slug:        project.Slug(),
		Description: project.Description(),
		WorkingDir:  project.WorkingDir(),
		Archived:    project.Archived(),
		Created:     project.CreatedAt(),
		Modified:    project.ModifiedAt(),
	}

	if project.Color() != nil {
		storage.Color = project.Color().String()
	}

	if len(project.Metadata()) > 0 {
		storage.Metadata = project.Metadata()
	}

	return storage
}

func ProjectFromStorage(storage *ProjectStorage) (*entity.Project, error) {
	if storage.ID == "" {
		return nil, fmt.Errorf("missing project ID")
	}
	if storage.Name == "" {
		return nil, fmt.Errorf("missing project name")
	}

	project, err := entity.NewProject(storage.ID, storage.Name, storage.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	if storage.Color != "" {
		color, err := valueobject.NewColor(storage.Color)
		if err == nil {
			project.SetColor(color)
		}
	}

	if storage.WorkingDir != "" {
		project.SetWorkingDir(storage.WorkingDir)
	}

	if storage.Archived {
		project.Archive()
	}

	for key, value := range storage.Metadata {
		project.SetMetadata(key, value)
	}

	return project, nil
}
