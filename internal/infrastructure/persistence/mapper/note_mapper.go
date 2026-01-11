package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"time"
)

type NoteStorage struct {
	ID          string            `yaml:"id"`
	ProjectID   string            `yaml:"project_id,omitempty"`
	Title       string            `yaml:"title"`
	NoteType    string            `yaml:"type"`
	Tags        []string          `yaml:"tags,omitempty"`
	LinkedTasks []string          `yaml:"linked_tasks,omitempty"`
	Date        time.Time         `yaml:"date"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	Created     time.Time         `yaml:"created"`
	Modified    time.Time         `yaml:"modified"`
}

func NoteToStorage(note *entity.Note) *NoteStorage {
	storage := &NoteStorage{
		ID:        note.ID(),
		ProjectID: note.ProjectID(),
		Title:     note.Title(),
		NoteType:  string(note.NoteType()),
		Tags:      note.Tags(),
		Date:      note.Date(),
		Created:   note.CreatedAt(),
		Modified:  note.ModifiedAt(),
	}

	linkedTasks := note.LinkedTasks()
	if len(linkedTasks) > 0 {
		storage.LinkedTasks = make([]string, len(linkedTasks))
		for i, t := range linkedTasks {
			storage.LinkedTasks[i] = t.String()
		}
	}

	if len(note.Metadata()) > 0 {
		storage.Metadata = note.Metadata()
	}

	return storage
}

func NoteFromStorage(storage *NoteStorage, content string) (*entity.Note, error) {
	if storage.ID == "" {
		return nil, fmt.Errorf("missing note ID")
	}

	noteType := entity.NoteType(storage.NoteType)
	if !noteType.IsValid() {
		noteType = entity.NoteTypeGeneral
	}

	note, err := entity.NewNote(storage.ID, storage.Title, noteType)
	if err != nil {
		return nil, err
	}

	if storage.ProjectID != "" {
		note.SetProjectID(storage.ProjectID)
	}

	note.SetContent(content)
	note.SetDate(storage.Date)

	for _, tag := range storage.Tags {
		note.AddTag(tag)
	}

	for _, taskIDStr := range storage.LinkedTasks {
		taskID, err := valueobject.ParseTaskID(taskIDStr)
		if err == nil {
			note.LinkTask(taskID)
		}
	}

	for key, value := range storage.Metadata {
		note.SetMetadata(key, value)
	}

	return note, nil
}
