package valueobject

import "fmt"

// Priority represents the priority level of a task
type Priority int

const (
	PriorityNone Priority = iota
	PriorityLow
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

var priorityNames = map[Priority]string{
	PriorityNone:     "none",
	PriorityLow:      "low",
	PriorityMedium:   "medium",
	PriorityHigh:     "high",
	PriorityCritical: "critical",
}

var priorityValues = map[string]Priority{
	"none":     PriorityNone,
	"low":      PriorityLow,
	"medium":   PriorityMedium,
	"high":     PriorityHigh,
	"critical": PriorityCritical,
}

// String returns the string representation of the priority
func (p Priority) String() string {
	if name, ok := priorityNames[p]; ok {
		return name
	}
	return "none"
}

// ParsePriority converts a string to a Priority
func ParsePriority(s string) (Priority, error) {
	if p, ok := priorityValues[s]; ok {
		return p, nil
	}
	return PriorityNone, fmt.Errorf("invalid priority: %s", s)
}

// IsValid checks if the priority is valid
func (p Priority) IsValid() bool {
	_, ok := priorityNames[p]
	return ok
}

// MarshalText implements encoding.TextMarshaler
func (p Priority) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (p *Priority) UnmarshalText(text []byte) error {
	priority, err := ParsePriority(string(text))
	if err != nil {
		return err
	}
	*p = priority
	return nil
}
