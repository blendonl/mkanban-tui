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
	"mkanban/internal/infrastructure/persistence/mapper"
	"mkanban/internal/infrastructure/serialization"
	"mkanban/pkg/filesystem"
)

const (
	noteMetadataFile = "metadata.yml"
	noteContentFile  = "content.md"
)

type NoteRepositoryImpl struct {
	pathBuilder *ProjectPathBuilder
}

func NewNoteRepository(rootPath string) repository.NoteRepository {
	return &NoteRepositoryImpl{
		pathBuilder: NewProjectPathBuilder(rootPath),
	}
}

func (r *NoteRepositoryImpl) Save(ctx context.Context, note *entity.Note) error {
	noteDir := r.getNoteDir(note)

	if err := filesystem.EnsureDir(noteDir, 0755); err != nil {
		return fmt.Errorf("failed to create note directory: %w", err)
	}

	storage := mapper.NoteToStorage(note)
	yamlData, err := serialization.SerializeYaml(storage)
	if err != nil {
		return fmt.Errorf("failed to serialize note metadata: %w", err)
	}

	metadataPath := filepath.Join(noteDir, noteMetadataFile)
	if err := filesystem.SafeWrite(metadataPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write note metadata: %w", err)
	}

	contentPath := filepath.Join(noteDir, noteContentFile)
	if err := filesystem.SafeWrite(contentPath, []byte(note.Content()), 0644); err != nil {
		return fmt.Errorf("failed to write note content: %w", err)
	}

	return nil
}

func (r *NoteRepositoryImpl) FindByID(ctx context.Context, id string) (*entity.Note, error) {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, entity.ErrNoteNotFound
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		notesDir := r.pathBuilder.ProjectNotesDir(entry.Name())
		note, err := r.findNoteInDir(notesDir, id)
		if err == nil {
			return note, nil
		}
	}

	globalNotesDir := r.pathBuilder.GlobalNotesDir()
	note, err := r.findNoteInDir(globalNotesDir, id)
	if err == nil {
		return note, nil
	}

	return nil, entity.ErrNoteNotFound
}

func (r *NoteRepositoryImpl) FindByProject(ctx context.Context, projectID string) ([]*entity.Note, error) {
	projectSlug, err := r.getProjectSlug(ctx, projectID)
	if err != nil {
		return nil, err
	}

	notesDir := r.pathBuilder.ProjectNotesDir(projectSlug)
	return r.loadAllNotesFromDir(notesDir)
}

func (r *NoteRepositoryImpl) FindByDate(ctx context.Context, projectID string, date time.Time) ([]*entity.Note, error) {
	notes, err := r.FindByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	dateStr := date.Format("2006-01-02")
	var result []*entity.Note
	for _, note := range notes {
		if note.Date().Format("2006-01-02") == dateStr {
			result = append(result, note)
		}
	}

	return result, nil
}

func (r *NoteRepositoryImpl) FindByDateRange(ctx context.Context, projectID string, start, end time.Time) ([]*entity.Note, error) {
	notes, err := r.FindByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var result []*entity.Note
	for _, note := range notes {
		noteDate := note.Date()
		if (noteDate.Equal(start) || noteDate.After(start)) && (noteDate.Equal(end) || noteDate.Before(end)) {
			result = append(result, note)
		}
	}

	return result, nil
}

func (r *NoteRepositoryImpl) FindByType(ctx context.Context, projectID string, noteType entity.NoteType) ([]*entity.Note, error) {
	notes, err := r.FindByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var result []*entity.Note
	for _, note := range notes {
		if note.NoteType() == noteType {
			result = append(result, note)
		}
	}

	return result, nil
}

func (r *NoteRepositoryImpl) FindByTag(ctx context.Context, projectID string, tag string) ([]*entity.Note, error) {
	notes, err := r.FindByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var result []*entity.Note
	for _, note := range notes {
		if note.HasTag(tag) {
			result = append(result, note)
		}
	}

	return result, nil
}

func (r *NoteRepositoryImpl) Search(ctx context.Context, projectID string, query string) ([]*entity.Note, error) {
	notes, err := r.FindByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var result []*entity.Note
	for _, note := range notes {
		if strings.Contains(strings.ToLower(note.Title()), queryLower) ||
			strings.Contains(strings.ToLower(note.Content()), queryLower) {
			result = append(result, note)
		}
	}

	return result, nil
}

func (r *NoteRepositoryImpl) Delete(ctx context.Context, id string) error {
	note, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	noteDir := r.getNoteDir(note)
	return filesystem.RemoveDir(noteDir)
}

func (r *NoteRepositoryImpl) FindGlobal(ctx context.Context) ([]*entity.Note, error) {
	globalNotesDir := r.pathBuilder.GlobalNotesDir()
	return r.loadAllNotesFromDir(globalNotesDir)
}

func (r *NoteRepositoryImpl) FindGlobalByDate(ctx context.Context, date time.Time) ([]*entity.Note, error) {
	notes, err := r.FindGlobal(ctx)
	if err != nil {
		return nil, err
	}

	dateStr := date.Format("2006-01-02")
	var result []*entity.Note
	for _, note := range notes {
		if note.Date().Format("2006-01-02") == dateStr {
			result = append(result, note)
		}
	}

	return result, nil
}

func (r *NoteRepositoryImpl) getNoteDir(note *entity.Note) string {
	dateStr := note.Date().Format("2006-01-02")
	slug := strings.ReplaceAll(strings.ToLower(note.Title()), " ", "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}

	if note.ProjectID() == "" {
		globalNotesDir := r.pathBuilder.GlobalNotesDir()
		return filepath.Join(globalNotesDir, dateStr, fmt.Sprintf("%s-%s", note.ID()[:8], slug))
	}

	projectSlug := r.getProjectSlugSync(note.ProjectID())
	if projectSlug == "" {
		projectSlug = note.ProjectID()
	}

	notesDir := r.pathBuilder.ProjectNotesDir(projectSlug)
	return filepath.Join(notesDir, dateStr, fmt.Sprintf("%s-%s", note.ID()[:8], slug))
}

func (r *NoteRepositoryImpl) getProjectSlug(ctx context.Context, projectID string) (string, error) {
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

		doc, err := serialization.ParseFrontmatter(data)
		if err != nil {
			continue
		}

		if doc.GetString("id") == projectID {
			return entry.Name(), nil
		}
	}

	return "", entity.ErrProjectNotFound
}

func (r *NoteRepositoryImpl) getProjectSlugSync(projectID string) string {
	projectsRoot := r.pathBuilder.ProjectsRoot()

	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		return ""
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

		doc, err := serialization.ParseFrontmatter(data)
		if err != nil {
			continue
		}

		if doc.GetString("id") == projectID {
			return entry.Name()
		}
	}

	return ""
}

func (r *NoteRepositoryImpl) findNoteInDir(notesDir string, id string) (*entity.Note, error) {
	err := filepath.WalkDir(notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() != filepath.Base(notesDir) {
			if strings.HasPrefix(d.Name(), id[:8]) {
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil {
		return nil, entity.ErrNoteNotFound
	}

	notes, err := r.loadAllNotesFromDir(notesDir)
	if err != nil {
		return nil, err
	}

	for _, note := range notes {
		if note.ID() == id {
			return note, nil
		}
	}

	return nil, entity.ErrNoteNotFound
}

func (r *NoteRepositoryImpl) loadAllNotesFromDir(notesDir string) ([]*entity.Note, error) {
	exists, err := filesystem.Exists(notesDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []*entity.Note{}, nil
	}

	var notes []*entity.Note

	err = filepath.WalkDir(notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != noteMetadataFile {
			return nil
		}

		noteDir := filepath.Dir(path)
		note, err := r.loadNote(noteDir)
		if err != nil {
			return nil
		}

		notes = append(notes, note)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return notes, nil
}

func (r *NoteRepositoryImpl) loadNote(noteDir string) (*entity.Note, error) {
	metadataPath := filepath.Join(noteDir, noteMetadataFile)
	contentPath := filepath.Join(noteDir, noteContentFile)

	metaData, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var storage mapper.NoteStorage
	if err := serialization.ParseYaml(metaData, &storage); err != nil {
		return nil, err
	}

	content := ""
	if contentData, err := os.ReadFile(contentPath); err == nil {
		content = string(contentData)
	}

	return mapper.NoteFromStorage(&storage, content)
}
