package valueobject

// ActionScope represents the scope at which an action applies
type ActionScope string

const (
	// ActionScopeGlobal applies to all boards
	ActionScopeGlobal ActionScope = "global"
	// ActionScopeBoard applies to a specific board
	ActionScopeBoard ActionScope = "board"
	// ActionScopeColumn applies to a specific column
	ActionScopeColumn ActionScope = "column"
	// ActionScopeTask applies to a specific task
	ActionScopeTask ActionScope = "task"
)

// IsValid checks if the action scope is valid
func (s ActionScope) IsValid() bool {
	switch s {
	case ActionScopeGlobal, ActionScopeBoard, ActionScopeColumn, ActionScopeTask:
		return true
	default:
		return false
	}
}

// String returns the string representation of the action scope
func (s ActionScope) String() string {
	return string(s)
}
