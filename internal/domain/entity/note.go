package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

type NoteType string

const (
	NoteTypeGeneral  NoteType = "general"
	NoteTypeJournal  NoteType = "journal"
	NoteTypeMeeting  NoteType = "meeting"
	NoteTypeStandup  NoteType = "standup"
	NoteTypeRetro    NoteType = "retrospective"
)

func (n NoteType) IsValid() bool {
	switch n {
	case NoteTypeGeneral, NoteTypeJournal, NoteTypeMeeting, NoteTypeStandup, NoteTypeRetro:
		return true
	}
	return false
}

type Note struct {
	id          string
	projectID   string
	title       string
	content     string
	noteType    NoteType
	tags        []string
	linkedTasks []*valueobject.TaskID
	date        time.Time
	metadata    map[string]string
	createdAt   time.Time
	modifiedAt  time.Time
}

func NewNote(id string, title string, noteType NoteType) (*Note, error) {
	if id == "" {
		return nil, ErrInvalidNoteID
	}
	if title == "" {
		return nil, ErrEmptyNoteTitle
	}
	if !noteType.IsValid() {
		noteType = NoteTypeGeneral
	}

	now := time.Now()
	return &Note{
		id:          id,
		title:       title,
		noteType:    noteType,
		tags:        make([]string, 0),
		linkedTasks: make([]*valueobject.TaskID, 0),
		date:        now,
		metadata:    make(map[string]string),
		createdAt:   now,
		modifiedAt:  now,
	}, nil
}

func (n *Note) ID() string {
	return n.id
}

func (n *Note) ProjectID() string {
	return n.projectID
}

func (n *Note) SetProjectID(projectID string) {
	n.projectID = projectID
	n.modifiedAt = time.Now()
}

func (n *Note) Title() string {
	return n.title
}

func (n *Note) UpdateTitle(title string) error {
	if title == "" {
		return ErrEmptyNoteTitle
	}
	n.title = title
	n.modifiedAt = time.Now()
	return nil
}

func (n *Note) Content() string {
	return n.content
}

func (n *Note) SetContent(content string) {
	n.content = content
	n.modifiedAt = time.Now()
}

func (n *Note) NoteType() NoteType {
	return n.noteType
}

func (n *Note) SetNoteType(noteType NoteType) {
	if noteType.IsValid() {
		n.noteType = noteType
		n.modifiedAt = time.Now()
	}
}

func (n *Note) Tags() []string {
	tagsCopy := make([]string, len(n.tags))
	copy(tagsCopy, n.tags)
	return tagsCopy
}

func (n *Note) AddTag(tag string) {
	for _, t := range n.tags {
		if t == tag {
			return
		}
	}
	n.tags = append(n.tags, tag)
	n.modifiedAt = time.Now()
}

func (n *Note) RemoveTag(tag string) {
	for i, t := range n.tags {
		if t == tag {
			n.tags = append(n.tags[:i], n.tags[i+1:]...)
			n.modifiedAt = time.Now()
			return
		}
	}
}

func (n *Note) HasTag(tag string) bool {
	for _, t := range n.tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (n *Note) LinkedTasks() []*valueobject.TaskID {
	tasksCopy := make([]*valueobject.TaskID, len(n.linkedTasks))
	copy(tasksCopy, n.linkedTasks)
	return tasksCopy
}

func (n *Note) LinkTask(taskID *valueobject.TaskID) {
	for _, t := range n.linkedTasks {
		if t.Equal(taskID) {
			return
		}
	}
	n.linkedTasks = append(n.linkedTasks, taskID)
	n.modifiedAt = time.Now()
}

func (n *Note) UnlinkTask(taskID *valueobject.TaskID) {
	for i, t := range n.linkedTasks {
		if t.Equal(taskID) {
			n.linkedTasks = append(n.linkedTasks[:i], n.linkedTasks[i+1:]...)
			n.modifiedAt = time.Now()
			return
		}
	}
}

func (n *Note) Date() time.Time {
	return n.date
}

func (n *Note) SetDate(date time.Time) {
	n.date = date
	n.modifiedAt = time.Now()
}

func (n *Note) Metadata() map[string]string {
	if n.metadata == nil {
		return make(map[string]string)
	}
	metadataCopy := make(map[string]string, len(n.metadata))
	for k, v := range n.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

func (n *Note) SetMetadata(key, value string) {
	if n.metadata == nil {
		n.metadata = make(map[string]string)
	}
	n.metadata[key] = value
	n.modifiedAt = time.Now()
}

func (n *Note) GetMetadata(key string) (string, bool) {
	if n.metadata == nil {
		return "", false
	}
	value, exists := n.metadata[key]
	return value, exists
}

func (n *Note) RemoveMetadata(key string) {
	if n.metadata == nil {
		return
	}
	delete(n.metadata, key)
	n.modifiedAt = time.Now()
}

func (n *Note) CreatedAt() time.Time {
	return n.createdAt
}

func (n *Note) ModifiedAt() time.Time {
	return n.modifiedAt
}

func (n *Note) IsJournal() bool {
	return n.noteType == NoteTypeJournal
}

func (n *Note) IsMeeting() bool {
	return n.noteType == NoteTypeMeeting
}
