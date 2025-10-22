package entity

import "time"

// Session represents a terminal multiplexer session
type Session struct {
	name        string
	workingDir  string
	sessionType string
	metadata    map[string]string
	createdAt   time.Time
	modifiedAt  time.Time
}

// NewSession creates a new Session entity
func NewSession(name, workingDir, sessionType string) (*Session, error) {
	if name == "" {
		return nil, ErrEmptySessionName
	}
	if workingDir == "" {
		return nil, ErrEmptyWorkingDir
	}
	if sessionType == "" {
		return nil, ErrInvalidSessionType
	}

	now := time.Now()
	return &Session{
		name:        name,
		workingDir:  workingDir,
		sessionType: sessionType,
		metadata:    make(map[string]string),
		createdAt:   now,
		modifiedAt:  now,
	}, nil
}

// Name returns the session name
func (s *Session) Name() string {
	return s.name
}

// WorkingDir returns the session's working directory
func (s *Session) WorkingDir() string {
	return s.workingDir
}

// SessionType returns the type of session (e.g., "tmux", "zellij")
func (s *Session) SessionType() string {
	return s.sessionType
}

// CreatedAt returns when the session was created
func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

// ModifiedAt returns when the session was last modified
func (s *Session) ModifiedAt() time.Time {
	return s.modifiedAt
}

// Metadata returns a copy of the session metadata
func (s *Session) Metadata() map[string]string {
	if s.metadata == nil {
		return make(map[string]string)
	}
	metadataCopy := make(map[string]string, len(s.metadata))
	for k, v := range s.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

// GetMetadata retrieves a specific metadata value by key
func (s *Session) GetMetadata(key string) (string, bool) {
	if s.metadata == nil {
		return "", false
	}
	value, exists := s.metadata[key]
	return value, exists
}

// SetMetadata sets a metadata key-value pair
func (s *Session) SetMetadata(key, value string) {
	if s.metadata == nil {
		s.metadata = make(map[string]string)
	}
	s.metadata[key] = value
	s.modifiedAt = time.Now()
}

// HasMetadata checks if a metadata key exists
func (s *Session) HasMetadata(key string) bool {
	if s.metadata == nil {
		return false
	}
	_, exists := s.metadata[key]
	return exists
}

// RemoveMetadata removes a metadata key
func (s *Session) RemoveMetadata(key string) {
	if s.metadata == nil {
		return
	}
	delete(s.metadata, key)
	s.modifiedAt = time.Now()
}

// UpdateWorkingDir updates the working directory
func (s *Session) UpdateWorkingDir(workingDir string) error {
	if workingDir == "" {
		return ErrEmptyWorkingDir
	}
	s.workingDir = workingDir
	s.modifiedAt = time.Now()
	return nil
}
