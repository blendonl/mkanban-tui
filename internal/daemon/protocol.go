package daemon

import "mkanban/internal/application/dto"

// Request types
const (
	RequestGetBoard        = "get_board"
	RequestListBoards      = "list_boards"
	RequestCreateBoard     = "create_board"
	RequestAddTask         = "add_task"
	RequestMoveTask        = "move_task"
	RequestUpdateTask      = "update_task"
	RequestDeleteTask      = "delete_task"
	RequestAddColumn       = "add_column"
	RequestDeleteColumn    = "delete_column"
	RequestGetActiveBoard  = "get_active_board"

	// Action request types
	RequestCreateAction    = "create_action"
	RequestUpdateAction    = "update_action"
	RequestDeleteAction    = "delete_action"
	RequestGetAction       = "get_action"
	RequestListActions     = "list_actions"
	RequestEnableAction    = "enable_action"
	RequestDisableAction   = "disable_action"
)

// Request represents a client request to the daemon
type Request struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Response represents a daemon response to the client
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GetBoardPayload contains data for getting a specific board
type GetBoardPayload struct {
	BoardID string `json:"board_id"`
}

// CreateBoardPayload contains data for creating a board
type CreateBoardPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AddTaskPayload contains data for adding a task
type AddTaskPayload struct {
	BoardID     string                `json:"board_id"`
	TaskRequest dto.CreateTaskRequest `json:"task"`
}

// MoveTaskPayload contains data for moving a task
type MoveTaskPayload struct {
	BoardID          string `json:"board_id"`
	TaskID           string `json:"task_id"`
	TargetColumnName string `json:"target_column_name"`
}

// UpdateTaskPayload contains data for updating a task
type UpdateTaskPayload struct {
	BoardID     string                `json:"board_id"`
	TaskID      string                `json:"task_id"`
	TaskRequest dto.UpdateTaskRequest `json:"task"`
}

// DeleteTaskPayload contains data for deleting a task
type DeleteTaskPayload struct {
	BoardID string `json:"board_id"`
	TaskID  string `json:"task_id"`
}

// AddColumnPayload contains data for adding a column
type AddColumnPayload struct {
	BoardID       string                  `json:"board_id"`
	ColumnRequest dto.CreateColumnRequest `json:"column"`
}

// DeleteColumnPayload contains data for deleting a column
type DeleteColumnPayload struct {
	BoardID    string `json:"board_id"`
	ColumnName string `json:"column_name"`
}

// CreateActionPayload contains data for creating an action
type CreateActionPayload struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Scope       string                 `json:"scope"`
	ScopeID     string                 `json:"scope_id"`
	Trigger     map[string]interface{} `json:"trigger"`
	ActionType  map[string]interface{} `json:"action_type"`
	Conditions  []map[string]interface{} `json:"conditions,omitempty"`
}

// UpdateActionPayload contains data for updating an action
type UpdateActionPayload struct {
	ActionID    string                   `json:"action_id"`
	Name        *string                  `json:"name,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Trigger     map[string]interface{}   `json:"trigger,omitempty"`
	ActionType  map[string]interface{}   `json:"action_type,omitempty"`
	Conditions  []map[string]interface{} `json:"conditions,omitempty"`
}

// DeleteActionPayload contains data for deleting an action
type DeleteActionPayload struct {
	ActionID string `json:"action_id"`
}

// GetActionPayload contains data for getting an action
type GetActionPayload struct {
	ActionID string `json:"action_id"`
}

// ListActionsPayload contains data for listing actions
type ListActionsPayload struct {
	Scope       *string `json:"scope,omitempty"`
	ScopeID     string  `json:"scope_id,omitempty"`
	EnabledOnly bool    `json:"enabled_only,omitempty"`
	TriggerType *string `json:"trigger_type,omitempty"`
}

// EnableActionPayload contains data for enabling an action
type EnableActionPayload struct {
	ActionID string `json:"action_id"`
}

// DisableActionPayload contains data for disabling an action
type DisableActionPayload struct {
	ActionID string `json:"action_id"`
}
