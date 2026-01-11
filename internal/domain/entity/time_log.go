package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

type TimeLogSource string

const (
	TimeLogSourceManual TimeLogSource = "manual"
	TimeLogSourceTimer  TimeLogSource = "timer"
	TimeLogSourceGit    TimeLogSource = "git"
	TimeLogSourceTmux   TimeLogSource = "tmux"
)

func (s TimeLogSource) IsValid() bool {
	switch s {
	case TimeLogSourceManual, TimeLogSourceTimer, TimeLogSourceGit, TimeLogSourceTmux:
		return true
	}
	return false
}

func (s TimeLogSource) String() string {
	return string(s)
}

type TimeLog struct {
	id          string
	projectID   string
	taskID      *valueobject.TaskID
	source      TimeLogSource
	startTime   time.Time
	endTime     *time.Time
	duration    time.Duration
	description string
	metadata    map[string]string
	createdAt   time.Time
	modifiedAt  time.Time
}

func NewTimeLog(
	id string,
	projectID string,
	source TimeLogSource,
	startTime time.Time,
) (*TimeLog, error) {
	if id == "" {
		return nil, ErrInvalidTimeLogID
	}
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if !source.IsValid() {
		return nil, ErrInvalidTimeLogSource
	}

	now := time.Now()
	return &TimeLog{
		id:         id,
		projectID:  projectID,
		source:     source,
		startTime:  startTime,
		metadata:   make(map[string]string),
		createdAt:  now,
		modifiedAt: now,
	}, nil
}

func (t *TimeLog) ID() string {
	return t.id
}

func (t *TimeLog) ProjectID() string {
	return t.projectID
}

func (t *TimeLog) TaskID() *valueobject.TaskID {
	return t.taskID
}

func (t *TimeLog) Source() TimeLogSource {
	return t.source
}

func (t *TimeLog) StartTime() time.Time {
	return t.startTime
}

func (t *TimeLog) EndTime() *time.Time {
	if t.endTime == nil {
		return nil
	}
	endCopy := *t.endTime
	return &endCopy
}

func (t *TimeLog) Duration() time.Duration {
	if t.endTime != nil {
		return t.endTime.Sub(t.startTime)
	}
	if t.duration > 0 {
		return t.duration
	}
	return time.Since(t.startTime)
}

func (t *TimeLog) Description() string {
	return t.description
}

func (t *TimeLog) Metadata() map[string]string {
	if t.metadata == nil {
		return make(map[string]string)
	}
	metadataCopy := make(map[string]string, len(t.metadata))
	for k, v := range t.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

func (t *TimeLog) CreatedAt() time.Time {
	return t.createdAt
}

func (t *TimeLog) ModifiedAt() time.Time {
	return t.modifiedAt
}

func (t *TimeLog) IsRunning() bool {
	return t.endTime == nil && t.duration == 0
}

func (t *TimeLog) SetTaskID(taskID *valueobject.TaskID) {
	t.taskID = taskID
	t.modifiedAt = time.Now()
}

func (t *TimeLog) SetDescription(description string) {
	t.description = description
	t.modifiedAt = time.Now()
}

func (t *TimeLog) Stop(endTime time.Time) error {
	if !t.IsRunning() {
		return ErrTimeLogAlreadyStopped
	}
	if endTime.Before(t.startTime) {
		return ErrInvalidEndTime
	}

	t.endTime = &endTime
	t.duration = endTime.Sub(t.startTime)
	t.modifiedAt = time.Now()
	return nil
}

func (t *TimeLog) SetDuration(duration time.Duration) error {
	if duration < 0 {
		return ErrInvalidDuration
	}

	t.duration = duration
	if t.endTime == nil {
		endTime := t.startTime.Add(duration)
		t.endTime = &endTime
	}
	t.modifiedAt = time.Now()
	return nil
}

func (t *TimeLog) SetMetadata(key, value string) {
	if t.metadata == nil {
		t.metadata = make(map[string]string)
	}
	t.metadata[key] = value
	t.modifiedAt = time.Now()
}

func (t *TimeLog) GetMetadata(key string) (string, bool) {
	if t.metadata == nil {
		return "", false
	}
	value, exists := t.metadata[key]
	return value, exists
}

func (t *TimeLog) RemoveMetadata(key string) {
	if t.metadata == nil {
		return
	}
	delete(t.metadata, key)
	t.modifiedAt = time.Now()
}

func NewTimeLogWithDuration(
	id string,
	projectID string,
	taskID *string,
	source TimeLogSource,
	startTime time.Time,
	endTime time.Time,
	description string,
) *TimeLog {
	now := time.Now()
	var taskIDVal *valueobject.TaskID
	if taskID != nil {
		parsed, err := valueobject.ParseTaskID(*taskID)
		if err == nil {
			taskIDVal = parsed
		}
	}
	return &TimeLog{
		id:          id,
		projectID:   projectID,
		taskID:      taskIDVal,
		source:      source,
		startTime:   startTime,
		endTime:     &endTime,
		duration:    endTime.Sub(startTime),
		description: description,
		metadata:    make(map[string]string),
		createdAt:   now,
		modifiedAt:  now,
	}
}
