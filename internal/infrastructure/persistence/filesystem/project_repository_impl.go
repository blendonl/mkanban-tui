package filesystem

import (
	"context"
	"fmt"
	"os"
	"time"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/persistence/mapper"
	"mkanban/internal/infrastructure/serialization"
	"mkanban/pkg/filesystem"
)

type ProjectRepositoryImpl struct {
	pathBuilder *ProjectPathBuilder
}

func NewProjectRepository(rootPath string) repository.ProjectRepository {
	return &ProjectRepositoryImpl{
		pathBuilder: NewProjectPathBuilder(rootPath),
	}
}

func (r *ProjectRepositoryImpl) Save(ctx context.Context, project *entity.Project) error {
	projectDir := r.pathBuilder.ProjectDir(project.Slug())

	if err := filesystem.EnsureDir(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	frontmatter := map[string]interface{}{
		"id":          project.ID(),
		"slug":        project.Slug(),
		"name":        project.Name(),
		"working_dir": project.WorkingDir(),
		"archived":    project.Archived(),
		"created":     project.CreatedAt().Format(time.RFC3339),
		"modified":    project.ModifiedAt().Format(time.RFC3339),
	}
	if project.Color() != nil {
		frontmatter["color"] = project.Color().String()
	}
	if len(project.Metadata()) > 0 {
		frontmatter["metadata"] = project.Metadata()
	}

	projectBody := serialization.SerializeMarkdownWithTitle(project.Name(), project.Description())
	projectDoc, err := serialization.SerializeFrontmatter(frontmatter, string(projectBody))
	if err != nil {
		return fmt.Errorf("failed to serialize project: %w", err)
	}

	metadataPath := r.pathBuilder.ProjectMetadata(project.Slug())
	if err := filesystem.SafeWrite(metadataPath, projectDoc, 0644); err != nil {
		return fmt.Errorf("failed to write project.md: %w", err)
	}

	if err := filesystem.EnsureDir(r.pathBuilder.ProjectBoardsDir(project.Slug()), 0755); err != nil {
		return fmt.Errorf("failed to create boards directory: %w", err)
	}

	if err := filesystem.EnsureDir(r.pathBuilder.ProjectNotesDir(project.Slug()), 0755); err != nil {
		return fmt.Errorf("failed to create notes directory: %w", err)
	}

	if err := filesystem.EnsureDir(r.pathBuilder.ProjectTimeDir(project.Slug()), 0755); err != nil {
		return fmt.Errorf("failed to create time directory: %w", err)
	}

	return nil
}

func (r *ProjectRepositoryImpl) FindByID(ctx context.Context, id string) (*entity.Project, error) {
	projects, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.ID() == id {
			return project, nil
		}
	}

	return nil, entity.ErrProjectNotFound
}

func (r *ProjectRepositoryImpl) FindBySlug(ctx context.Context, slug string) (*entity.Project, error) {
	projectDir := r.pathBuilder.ProjectDir(slug)

	exists, err := filesystem.Exists(projectDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, entity.ErrProjectNotFound
	}

	return r.loadProject(slug)
}

func (r *ProjectRepositoryImpl) FindAll(ctx context.Context) ([]*entity.Project, error) {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	if err := filesystem.EnsureDir(projectsRoot, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	projects := make([]*entity.Project, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		project, err := r.loadProject(entry.Name())
		if err != nil {
			continue
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (r *ProjectRepositoryImpl) Delete(ctx context.Context, id string) error {
	project, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	projectDir := r.pathBuilder.ProjectDir(project.Slug())
	return filesystem.RemoveDir(projectDir)
}

func (r *ProjectRepositoryImpl) Exists(ctx context.Context, id string) (bool, error) {
	project, err := r.FindByID(ctx, id)
	if err == entity.ErrProjectNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return project != nil, nil
}

func (r *ProjectRepositoryImpl) loadProject(slug string) (*entity.Project, error) {
	metadataPath := r.pathBuilder.ProjectMetadata(slug)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project.md: %w", err)
	}

	doc, err := serialization.ParseFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project.md: %w", err)
	}

	markdownDoc, err := serialization.ParseMarkdownWithTitle([]byte(doc.Content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse project.md content: %w", err)
	}

	archived := false
	if archivedRaw, ok := doc.Frontmatter["archived"]; ok {
		switch value := archivedRaw.(type) {
		case bool:
			archived = value
		case string:
			archived = value == "true"
		}
	}

	storage := mapper.ProjectStorage{
		ID:          doc.GetString("id"),
		Name:        markdownDoc.Title,
		Slug:        doc.GetString("slug"),
		Description: markdownDoc.Content,
		WorkingDir:  doc.GetString("working_dir"),
		Archived:    archived,
		Metadata:    map[string]string{},
	}

	if storage.Name == "" {
		storage.Name = doc.GetString("name")
	}
	if storage.Name == "" {
		storage.Name = slug
	}
	if storage.Slug == "" {
		storage.Slug = valueobject.GenerateSlug(storage.Name)
	}

	if color := doc.GetString("color"); color != "" {
		storage.Color = color
	}

	if metadataRaw, ok := doc.Frontmatter["metadata"]; ok {
		if metadataMap, ok := metadataRaw.(map[string]interface{}); ok {
			for key, value := range metadataMap {
				if strValue, ok := value.(string); ok {
					storage.Metadata[key] = strValue
				}
			}
		}
	}

	return mapper.ProjectFromStorage(&storage)
}
