package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

type Project struct {
	id          string
	name        string
	slug        string
	description string
	boards      []*Board
	color       *valueobject.Color
	archived    bool
	workingDir  string
	createdAt   time.Time
	modifiedAt  time.Time
	metadata    map[string]string
}

func NewProject(id string, name string, description string) (*Project, error) {
	if id == "" {
		return nil, ErrInvalidProjectID
	}
	if name == "" {
		return nil, ErrEmptyProjectName
	}

	slug := valueobject.GenerateSlug(name)
	now := time.Now()

	return &Project{
		id:          id,
		name:        name,
		slug:        slug,
		description: description,
		boards:      make([]*Board, 0),
		archived:    false,
		createdAt:   now,
		modifiedAt:  now,
		metadata:    make(map[string]string),
	}, nil
}

func (p *Project) ID() string {
	return p.id
}

func (p *Project) Name() string {
	return p.name
}

func (p *Project) Slug() string {
	return p.slug
}

func (p *Project) Description() string {
	return p.description
}

func (p *Project) Boards() []*Board {
	boardsCopy := make([]*Board, len(p.boards))
	copy(boardsCopy, p.boards)
	return boardsCopy
}

func (p *Project) Color() *valueobject.Color {
	return p.color
}

func (p *Project) Archived() bool {
	return p.archived
}

func (p *Project) WorkingDir() string {
	return p.workingDir
}

func (p *Project) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Project) ModifiedAt() time.Time {
	return p.modifiedAt
}

func (p *Project) Metadata() map[string]string {
	if p.metadata == nil {
		return make(map[string]string)
	}
	metadataCopy := make(map[string]string, len(p.metadata))
	for k, v := range p.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

func (p *Project) UpdateName(name string) error {
	if name == "" {
		return ErrEmptyProjectName
	}
	p.name = name
	p.slug = valueobject.GenerateSlug(name)
	p.modifiedAt = time.Now()
	return nil
}

func (p *Project) UpdateDescription(description string) {
	p.description = description
	p.modifiedAt = time.Now()
}

func (p *Project) SetColor(color *valueobject.Color) {
	p.color = color
	p.modifiedAt = time.Now()
}

func (p *Project) SetWorkingDir(dir string) {
	p.workingDir = dir
	p.modifiedAt = time.Now()
}

func (p *Project) Archive() {
	p.archived = true
	p.modifiedAt = time.Now()
}

func (p *Project) Unarchive() {
	p.archived = false
	p.modifiedAt = time.Now()
}

func (p *Project) AddBoard(board *Board) error {
	if board == nil {
		return ErrBoardNotFound
	}

	for _, existingBoard := range p.boards {
		if existingBoard.ID() == board.ID() {
			return ErrBoardAlreadyExists
		}
	}

	p.boards = append(p.boards, board)
	p.modifiedAt = time.Now()
	return nil
}

func (p *Project) RemoveBoard(boardID string) (*Board, error) {
	for i, board := range p.boards {
		if board.ID() == boardID {
			p.boards = append(p.boards[:i], p.boards[i+1:]...)
			p.modifiedAt = time.Now()
			return board, nil
		}
	}
	return nil, ErrBoardNotFound
}

func (p *Project) GetBoard(boardID string) (*Board, error) {
	for _, board := range p.boards {
		if board.ID() == boardID || board.Name() == boardID {
			return board, nil
		}
	}
	return nil, ErrBoardNotFound
}

func (p *Project) BoardCount() int {
	return len(p.boards)
}

func (p *Project) TotalTaskCount() int {
	total := 0
	for _, board := range p.boards {
		total += board.TotalTaskCount()
	}
	return total
}

func (p *Project) SetMetadata(key, value string) {
	if p.metadata == nil {
		p.metadata = make(map[string]string)
	}
	p.metadata[key] = value
	p.modifiedAt = time.Now()
}

func (p *Project) GetMetadata(key string) (string, bool) {
	if p.metadata == nil {
		return "", false
	}
	value, exists := p.metadata[key]
	return value, exists
}

func (p *Project) RemoveMetadata(key string) {
	if p.metadata == nil {
		return
	}
	delete(p.metadata, key)
	p.modifiedAt = time.Now()
}
