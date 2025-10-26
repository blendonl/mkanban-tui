package entity

import (
	"mkanban/internal/domain/valueobject"
)

// ActionTypeEnum represents the type of action
type ActionTypeEnum string

const (
	ActionTypeNotification  ActionTypeEnum = "notification"
	ActionTypeScript        ActionTypeEnum = "script"
	ActionTypeTaskMutation  ActionTypeEnum = "task_mutation"
	ActionTypeTaskMovement  ActionTypeEnum = "task_movement"
	ActionTypeTaskCreation  ActionTypeEnum = "task_creation"
)

// ActionType defines the interface for different action types
type ActionType interface {
	Type() ActionTypeEnum
	Execute(ctx *ActionContext) error
	Validate() error
}

// ActionContext contains information needed to execute an action
type ActionContext struct {
	Task         *Task
	Column       *Column
	Board        *Board
	Event        *DomainEvent
	Notifier     Notifier
	ScriptRunner ScriptRunner
	TaskMutator  TaskMutator
}

// Notifier interface for sending notifications
type Notifier interface {
	SendNotification(title, message string, metadata map[string]string) error
}

// ScriptRunner interface for executing scripts
type ScriptRunner interface {
	RunScript(scriptPath string, env map[string]string) error
}

// TaskMutator interface for mutating tasks
type TaskMutator interface {
	UpdateTask(boardID string, task *Task) error
	MoveTask(boardID string, taskID *valueobject.TaskID, targetColumn string) error
	CreateTask(boardID string, columnName string, task *Task) error
}

// NotificationAction sends a notification
type NotificationAction struct {
	Title    string
	Message  string
	Metadata map[string]string
}

// NewNotificationAction creates a new notification action
func NewNotificationAction(title, message string, metadata map[string]string) *NotificationAction {
	return &NotificationAction{
		Title:    title,
		Message:  message,
		Metadata: metadata,
	}
}

// Type returns the action type
func (a *NotificationAction) Type() ActionTypeEnum {
	return ActionTypeNotification
}

// Execute sends the notification
func (a *NotificationAction) Execute(ctx *ActionContext) error {
	if ctx.Notifier == nil {
		return ErrNotifierNotAvailable
	}

	// Template replacement for title and message
	title := a.interpolateTemplate(a.Title, ctx)
	message := a.interpolateTemplate(a.Message, ctx)

	return ctx.Notifier.SendNotification(title, message, a.Metadata)
}

// Validate checks if the notification action is valid
func (a *NotificationAction) Validate() error {
	if a.Title == "" {
		return ErrInvalidNotificationTitle
	}
	if a.Message == "" {
		return ErrInvalidNotificationMessage
	}
	return nil
}

// interpolateTemplate replaces template variables in strings
func (a *NotificationAction) interpolateTemplate(template string, ctx *ActionContext) string {
	// Simple template replacement
	// TODO: Implement proper template engine
	result := template
	if ctx.Task != nil {
		// Replace common variables
		// result = strings.ReplaceAll(result, "{{task.title}}", ctx.Task.Title())
		// etc.
	}
	return result
}

// ScriptAction executes a custom script
type ScriptAction struct {
	ScriptPath string
	EnvVars    map[string]string
}

// NewScriptAction creates a new script action
func NewScriptAction(scriptPath string, envVars map[string]string) *ScriptAction {
	return &ScriptAction{
		ScriptPath: scriptPath,
		EnvVars:    envVars,
	}
}

// Type returns the action type
func (a *ScriptAction) Type() ActionTypeEnum {
	return ActionTypeScript
}

// Execute runs the script
func (a *ScriptAction) Execute(ctx *ActionContext) error {
	if ctx.ScriptRunner == nil {
		return ErrScriptRunnerNotAvailable
	}

	// Build environment variables with context
	env := make(map[string]string)
	for k, v := range a.EnvVars {
		env[k] = v
	}

	// Add context variables
	if ctx.Task != nil {
		env["TASK_ID"] = ctx.Task.ID().String()
		env["TASK_TITLE"] = ctx.Task.Title()
		env["TASK_PRIORITY"] = ctx.Task.Priority().String()
		env["TASK_STATUS"] = ctx.Task.Status().String()
	}
	if ctx.Board != nil {
		env["BOARD_ID"] = ctx.Board.ID()
		env["BOARD_NAME"] = ctx.Board.Name()
	}
	if ctx.Column != nil {
		env["COLUMN_NAME"] = ctx.Column.Name()
	}

	return ctx.ScriptRunner.RunScript(a.ScriptPath, env)
}

// Validate checks if the script action is valid
func (a *ScriptAction) Validate() error {
	if a.ScriptPath == "" {
		return ErrInvalidScriptPath
	}
	return nil
}

// TaskMutationAction updates task fields
type TaskMutationAction struct {
	UpdatePriority *valueobject.Priority
	UpdateStatus   *valueobject.Status
	AddTags        []string
	RemoveTags     []string
	SetMetadata    map[string]string
}

// NewTaskMutationAction creates a new task mutation action
func NewTaskMutationAction() *TaskMutationAction {
	return &TaskMutationAction{
		AddTags:     make([]string, 0),
		RemoveTags:  make([]string, 0),
		SetMetadata: make(map[string]string),
	}
}

// Type returns the action type
func (a *TaskMutationAction) Type() ActionTypeEnum {
	return ActionTypeTaskMutation
}

// Execute mutates the task
func (a *TaskMutationAction) Execute(ctx *ActionContext) error {
	if ctx.Task == nil {
		return ErrTaskNotFound
	}
	if ctx.TaskMutator == nil {
		return ErrTaskMutatorNotAvailable
	}

	// Apply mutations
	if a.UpdatePriority != nil {
		if err := ctx.Task.UpdatePriority(*a.UpdatePriority); err != nil {
			return err
		}
	}

	if a.UpdateStatus != nil {
		if err := ctx.Task.UpdateStatus(*a.UpdateStatus); err != nil {
			return err
		}
	}

	for _, tag := range a.AddTags {
		ctx.Task.AddTag(tag)
	}

	for _, tag := range a.RemoveTags {
		ctx.Task.RemoveTag(tag)
	}

	for key, value := range a.SetMetadata {
		ctx.Task.SetMetadata(key, value)
	}

	// Persist the changes
	return ctx.TaskMutator.UpdateTask(ctx.Board.ID(), ctx.Task)
}

// Validate checks if the task mutation action is valid
func (a *TaskMutationAction) Validate() error {
	if a.UpdatePriority != nil && !a.UpdatePriority.IsValid() {
		return ErrInvalidPriority
	}
	if a.UpdateStatus != nil && !a.UpdateStatus.IsValid() {
		return ErrInvalidStatus
	}
	return nil
}

// TaskMovementAction moves a task to another column
type TaskMovementAction struct {
	TargetColumn string
}

// NewTaskMovementAction creates a new task movement action
func NewTaskMovementAction(targetColumn string) *TaskMovementAction {
	return &TaskMovementAction{
		TargetColumn: targetColumn,
	}
}

// Type returns the action type
func (a *TaskMovementAction) Type() ActionTypeEnum {
	return ActionTypeTaskMovement
}

// Execute moves the task
func (a *TaskMovementAction) Execute(ctx *ActionContext) error {
	if ctx.Task == nil {
		return ErrTaskNotFound
	}
	if ctx.TaskMutator == nil {
		return ErrTaskMutatorNotAvailable
	}

	return ctx.TaskMutator.MoveTask(ctx.Board.ID(), ctx.Task.ID(), a.TargetColumn)
}

// Validate checks if the task movement action is valid
func (a *TaskMovementAction) Validate() error {
	if a.TargetColumn == "" {
		return ErrInvalidTargetColumn
	}
	return nil
}

// TaskCreationAction creates a new task
type TaskCreationAction struct {
	Title       string
	Description string
	Priority    valueobject.Priority
	Status      valueobject.Status
	ColumnName  string
	Tags        []string
	Metadata    map[string]string
}

// NewTaskCreationAction creates a new task creation action
func NewTaskCreationAction(title, description, columnName string) *TaskCreationAction {
	return &TaskCreationAction{
		Title:       title,
		Description: description,
		ColumnName:  columnName,
		Priority:    valueobject.PriorityMedium,
		Status:      valueobject.StatusTodo,
		Tags:        make([]string, 0),
		Metadata:    make(map[string]string),
	}
}

// Type returns the action type
func (a *TaskCreationAction) Type() ActionTypeEnum {
	return ActionTypeTaskCreation
}

// Execute creates the task
func (a *TaskCreationAction) Execute(ctx *ActionContext) error {
	if ctx.TaskMutator == nil {
		return ErrTaskMutatorNotAvailable
	}

	// Generate task ID
	taskID, err := ctx.Board.GenerateNextTaskID(a.Title)
	if err != nil {
		return err
	}

	// Create new task
	task, err := NewTask(taskID, a.Title, a.Description, a.Priority, a.Status)
	if err != nil {
		return err
	}

	// Add tags and metadata
	for _, tag := range a.Tags {
		task.AddTag(tag)
	}
	for key, value := range a.Metadata {
		task.SetMetadata(key, value)
	}

	return ctx.TaskMutator.CreateTask(ctx.Board.ID(), a.ColumnName, task)
}

// Validate checks if the task creation action is valid
func (a *TaskCreationAction) Validate() error {
	if a.Title == "" {
		return ErrEmptyTaskName
	}
	if a.ColumnName == "" {
		return ErrInvalidTargetColumn
	}
	if !a.Priority.IsValid() {
		return ErrInvalidPriority
	}
	if !a.Status.IsValid() {
		return ErrInvalidStatus
	}
	return nil
}
