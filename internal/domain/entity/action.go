package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

// Action represents an action/reminder that can be triggered
type Action struct {
	id          string
	name        string
	description string
	scope       valueobject.ActionScope
	scopeID     string // Board/Column/Task ID depending on scope
	enabled     bool
	trigger     Trigger
	actionType  ActionType
	conditions  *ConditionGroup
	createdAt   time.Time
	modifiedAt  time.Time
	lastRun     *time.Time
}

// NewAction creates a new Action entity
func NewAction(
	id string,
	name string,
	description string,
	scope valueobject.ActionScope,
	scopeID string,
	trigger Trigger,
	actionType ActionType,
	conditions *ConditionGroup,
) (*Action, error) {
	if id == "" {
		return nil, ErrInvalidActionID
	}
	if name == "" {
		return nil, ErrInvalidActionName
	}
	if !scope.IsValid() {
		return nil, ErrInvalidActionScope
	}
	if trigger == nil {
		return nil, ErrInvalidTrigger
	}
	if actionType == nil {
		return nil, ErrInvalidActionType
	}

	// Validate action type
	if err := actionType.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Action{
		id:          id,
		name:        name,
		description: description,
		scope:       scope,
		scopeID:     scopeID,
		enabled:     true,
		trigger:     trigger,
		actionType:  actionType,
		conditions:  conditions,
		createdAt:   now,
		modifiedAt:  now,
	}, nil
}

// ID returns the action ID
func (a *Action) ID() string {
	return a.id
}

// Name returns the action name
func (a *Action) Name() string {
	return a.name
}

// Description returns the action description
func (a *Action) Description() string {
	return a.description
}

// Scope returns the action scope
func (a *Action) Scope() valueobject.ActionScope {
	return a.scope
}

// ScopeID returns the scope ID (board/column/task ID)
func (a *Action) ScopeID() string {
	return a.scopeID
}

// Enabled returns whether the action is enabled
func (a *Action) Enabled() bool {
	return a.enabled
}

// Trigger returns the action trigger
func (a *Action) Trigger() Trigger {
	return a.trigger
}

// ActionType returns the action type
func (a *Action) ActionType() ActionType {
	return a.actionType
}

// Conditions returns the action conditions
func (a *Action) Conditions() *ConditionGroup {
	return a.conditions
}

// CreatedAt returns when the action was created
func (a *Action) CreatedAt() time.Time {
	return a.createdAt
}

// ModifiedAt returns when the action was last modified
func (a *Action) ModifiedAt() time.Time {
	return a.modifiedAt
}

// LastRun returns when the action was last executed
func (a *Action) LastRun() *time.Time {
	if a.lastRun == nil {
		return nil
	}
	lastRunCopy := *a.lastRun
	return &lastRunCopy
}

// UpdateName updates the action name
func (a *Action) UpdateName(name string) error {
	if name == "" {
		return ErrInvalidActionName
	}
	a.name = name
	a.modifiedAt = time.Now()
	return nil
}

// UpdateDescription updates the action description
func (a *Action) UpdateDescription(description string) {
	a.description = description
	a.modifiedAt = time.Now()
}

// Enable enables the action
func (a *Action) Enable() {
	a.enabled = true
	a.modifiedAt = time.Now()
}

// Disable disables the action
func (a *Action) Disable() {
	a.enabled = false
	a.modifiedAt = time.Now()
}

// UpdateTrigger updates the action trigger
func (a *Action) UpdateTrigger(trigger Trigger) error {
	if trigger == nil {
		return ErrInvalidTrigger
	}
	a.trigger = trigger
	a.modifiedAt = time.Now()
	return nil
}

// UpdateActionType updates the action type
func (a *Action) UpdateActionType(actionType ActionType) error {
	if actionType == nil {
		return ErrInvalidActionType
	}
	if err := actionType.Validate(); err != nil {
		return err
	}
	a.actionType = actionType
	a.modifiedAt = time.Now()
	return nil
}

// UpdateConditions updates the action conditions
func (a *Action) UpdateConditions(conditions *ConditionGroup) {
	a.conditions = conditions
	a.modifiedAt = time.Now()
}

// MarkAsRun marks the action as having been run
func (a *Action) MarkAsRun() {
	now := time.Now()
	a.lastRun = &now
	a.modifiedAt = now
}

// ShouldExecute checks if the action should execute given the context
func (a *Action) ShouldExecute(ctx *TriggerContext) bool {
	// Check if action is enabled
	if !a.enabled {
		return false
	}

	// Check if trigger should fire
	if !a.trigger.ShouldTrigger(ctx) {
		return false
	}

	// Check conditions if they exist
	if a.conditions != nil && ctx.Task != nil {
		if !a.conditions.Evaluate(ctx.Task, ctx.Column) {
			return false
		}
	}

	return true
}

// Execute executes the action
func (a *Action) Execute(actCtx *ActionContext) error {
	if !a.enabled {
		return ErrActionDisabled
	}

	if err := a.actionType.Execute(actCtx); err != nil {
		return err
	}

	a.MarkAsRun()
	return nil
}

// Clone creates a copy of the action (for copy-on-assign)
func (a *Action) Clone(newID string, newScopeID string, newScope valueobject.ActionScope) *Action {
	now := time.Now()
	return &Action{
		id:          newID,
		name:        a.name,
		description: a.description,
		scope:       newScope,
		scopeID:     newScopeID,
		enabled:     a.enabled,
		trigger:     a.trigger,
		actionType:  a.actionType,
		conditions:  a.conditions,
		createdAt:   now,
		modifiedAt:  now,
		lastRun:     nil, // Reset last run for cloned action
	}
}

// MatchesScope checks if the action applies to the given scope context
func (a *Action) MatchesScope(boardID, columnID, taskID string) bool {
	switch a.scope {
	case valueobject.ActionScopeGlobal:
		return true
	case valueobject.ActionScopeBoard:
		return a.scopeID == boardID
	case valueobject.ActionScopeColumn:
		return a.scopeID == columnID
	case valueobject.ActionScopeTask:
		return a.scopeID == taskID
	default:
		return false
	}
}
