package valueobject

import "fmt"

// Status represents the current status of a task
type Status int

const (
	StatusTodo Status = iota
	StatusInProgress
	StatusBlocked
	StatusInReview
	StatusDone
)

var statusNames = map[Status]string{
	StatusTodo:       "todo",
	StatusInProgress: "in_progress",
	StatusBlocked:    "blocked",
	StatusInReview:   "in_review",
	StatusDone:       "done",
}

var statusValues = map[string]Status{
	"todo":        StatusTodo,
	"in_progress": StatusInProgress,
	"blocked":     StatusBlocked,
	"in_review":   StatusInReview,
	"done":        StatusDone,
}

// String returns the string representation of the status
func (s Status) String() string {
	if name, ok := statusNames[s]; ok {
		return name
	}
	return "todo"
}

// ParseStatus converts a string to a Status
func ParseStatus(str string) (Status, error) {
	if s, ok := statusValues[str]; ok {
		return s, nil
	}
	return StatusTodo, fmt.Errorf("invalid status: %s", str)
}

// IsValid checks if the status is valid
func (s Status) IsValid() bool {
	_, ok := statusNames[s]
	return ok
}

// MarshalText implements encoding.TextMarshaler
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (s *Status) UnmarshalText(text []byte) error {
	status, err := ParseStatus(string(text))
	if err != nil {
		return err
	}
	*s = status
	return nil
}
