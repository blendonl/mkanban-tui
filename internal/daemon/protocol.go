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

	// Real-time update request types
	RequestSubscribe   = "subscribe"
	RequestUnsubscribe = "unsubscribe"
	RequestPing        = "ping"

	// Time tracking request types
	RequestStartTimer      = "start_timer"
	RequestStopTimer       = "stop_timer"
	RequestGetActiveTimers = "get_active_timers"
	RequestListTimeLogs    = "list_time_logs"
	RequestAddTimeEntry    = "add_time_entry"

	// Project request types
	RequestCreateProject = "create_project"
	RequestGetProject    = "get_project"
	RequestListProjects  = "list_projects"
	RequestUpdateProject = "update_project"
	RequestDeleteProject = "delete_project"

	// Agenda request types
	RequestScheduleTask  = "schedule_task"
	RequestCreateMeeting = "create_meeting"
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
	ProjectID   string `json:"project_id"`
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

// SubscribePayload contains data for subscribing to board updates
type SubscribePayload struct {
	BoardID string `json:"board_id"`
}

// UnsubscribePayload contains data for unsubscribing from board updates
type UnsubscribePayload struct {
	BoardID string `json:"board_id"`
}

// GetActiveBoardPayload contains data for getting the active board
type GetActiveBoardPayload struct {
	SessionName string `json:"session_name,omitempty"`
}

// Notification represents a push notification from daemon to client
type Notification struct {
	Type    string      `json:"type"`
	BoardID string      `json:"board_id,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Time tracking payloads

type StartTimerPayload struct {
	ProjectID   string  `json:"project_id"`
	TaskID      *string `json:"task_id,omitempty"`
	Description string  `json:"description,omitempty"`
}

type StopTimerPayload struct {
	ProjectID string  `json:"project_id"`
	TaskID    *string `json:"task_id,omitempty"`
}

type ListTimeLogsPayload struct {
	ProjectID string  `json:"project_id,omitempty"`
	TaskID    *string `json:"task_id,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

type AddTimeEntryPayload struct {
	ProjectID   string `json:"project_id"`
	TaskID      *string `json:"task_id,omitempty"`
	StartTime   string `json:"start_time"`
	Duration    int64  `json:"duration_seconds"`
	Description string `json:"description,omitempty"`
}

// Project payloads

type CreateProjectPayload struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	WorkingDir  string `json:"working_dir,omitempty"`
	Color       string `json:"color,omitempty"`
}

type GetProjectPayload struct {
	ProjectID string `json:"project_id"`
}

type UpdateProjectPayload struct {
	ProjectID   string  `json:"project_id"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	WorkingDir  *string `json:"working_dir,omitempty"`
	Color       *string `json:"color,omitempty"`
	Archived    *bool   `json:"archived,omitempty"`
}

type DeleteProjectPayload struct {
	ProjectID string `json:"project_id"`
}

// Agenda payloads

type ScheduleTaskPayload struct {
	TaskID   string  `json:"task_id"`
	Date     string  `json:"date"`
	Time     *string `json:"time,omitempty"`
	Duration *string `json:"duration,omitempty"`
}

type CreateMeetingPayload struct {
	BoardID   string   `json:"board_id"`
	Title     string   `json:"title"`
	Date      string   `json:"date"`
	Time      *string  `json:"time,omitempty"`
	Duration  *string  `json:"duration,omitempty"`
	Attendees []string `json:"attendees,omitempty"`
	Location  *string  `json:"location,omitempty"`
}

// Notification types
const (
	NotificationBoardUpdated = "board_updated"
	NotificationTaskCreated  = "task_created"
	NotificationTaskUpdated  = "task_updated"
	NotificationTaskMoved    = "task_moved"
	NotificationTaskDeleted  = "task_deleted"
	NotificationPong         = "pong"
)
