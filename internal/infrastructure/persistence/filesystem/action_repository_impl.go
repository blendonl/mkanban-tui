package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/config"
)

// ActionRepositoryImpl implements the ActionRepository interface
type ActionRepositoryImpl struct {
	config *config.Config
	mu     sync.RWMutex
}

// NewActionRepository creates a new action repository
func NewActionRepository(cfg *config.Config) repository.ActionRepository {
	return &ActionRepositoryImpl{
		config: cfg,
	}
}

// getActionsPath returns the path to the actions directory
func (r *ActionRepositoryImpl) getActionsPath() string {
	return filepath.Join(r.config.Storage.DataPath, "actions")
}

// getActionFilePath returns the path to an action file
func (r *ActionRepositoryImpl) getActionFilePath(id string) string {
	return filepath.Join(r.getActionsPath(), fmt.Sprintf("%s.yml", id))
}

// Create creates a new action
func (r *ActionRepositoryImpl) Create(ctx context.Context, action *entity.Action) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure actions directory exists
	if err := os.MkdirAll(r.getActionsPath(), 0755); err != nil {
		return fmt.Errorf("failed to create actions directory: %w", err)
	}

	// Check if action already exists
	filePath := r.getActionFilePath(action.ID())
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("action already exists: %s", action.ID())
	}

	// Convert action to storage model
	storageAction := r.toStorageAction(action)

	// Marshal to YAML
	data, err := yaml.Marshal(storageAction)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write action file: %w", err)
	}

	return nil
}

// Update updates an existing action
func (r *ActionRepositoryImpl) Update(ctx context.Context, action *entity.Action) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filePath := r.getActionFilePath(action.ID())

	// Check if action exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return entity.ErrActionNotFound
	}

	// Convert action to storage model
	storageAction := r.toStorageAction(action)

	// Marshal to YAML
	data, err := yaml.Marshal(storageAction)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write action file: %w", err)
	}

	return nil
}

// Delete deletes an action by ID
func (r *ActionRepositoryImpl) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filePath := r.getActionFilePath(id)

	// Check if action exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return entity.ErrActionNotFound
	}

	// Delete file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete action file: %w", err)
	}

	return nil
}

// GetByID retrieves an action by ID
func (r *ActionRepositoryImpl) GetByID(ctx context.Context, id string) (*entity.Action, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filePath := r.getActionFilePath(id)

	// Check if action exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, entity.ErrActionNotFound
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read action file: %w", err)
	}

	// Unmarshal from YAML
	var storageAction StorageAction
	if err := yaml.Unmarshal(data, &storageAction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action: %w", err)
	}

	// Convert to domain entity
	return r.fromStorageAction(&storageAction)
}

// ListAll retrieves all actions
func (r *ActionRepositoryImpl) ListAll(ctx context.Context) ([]*entity.Action, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actionsPath := r.getActionsPath()

	// Check if actions directory exists
	if _, err := os.Stat(actionsPath); os.IsNotExist(err) {
		return []*entity.Action{}, nil
	}

	// Read all action files
	files, err := filepath.Glob(filepath.Join(actionsPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list action files: %w", err)
	}

	actions := make([]*entity.Action, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var storageAction StorageAction
		if err := yaml.Unmarshal(data, &storageAction); err != nil {
			continue // Skip invalid files
		}

		action, err := r.fromStorageAction(&storageAction)
		if err != nil {
			continue // Skip invalid actions
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// ListByScope retrieves actions for a specific scope
func (r *ActionRepositoryImpl) ListByScope(ctx context.Context, scope valueobject.ActionScope, scopeID string) ([]*entity.Action, error) {
	allActions, err := r.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*entity.Action, 0)
	for _, action := range allActions {
		if action.Scope() == scope && action.ScopeID() == scopeID {
			filtered = append(filtered, action)
		}
	}

	return filtered, nil
}

// ListGlobal retrieves all global actions
func (r *ActionRepositoryImpl) ListGlobal(ctx context.Context) ([]*entity.Action, error) {
	return r.ListByScope(ctx, valueobject.ActionScopeGlobal, "")
}

// ListByBoard retrieves actions for a specific board
func (r *ActionRepositoryImpl) ListByBoard(ctx context.Context, boardID string) ([]*entity.Action, error) {
	return r.ListByScope(ctx, valueobject.ActionScopeBoard, boardID)
}

// ListByColumn retrieves actions for a specific column
func (r *ActionRepositoryImpl) ListByColumn(ctx context.Context, columnID string) ([]*entity.Action, error) {
	return r.ListByScope(ctx, valueobject.ActionScopeColumn, columnID)
}

// ListByTask retrieves actions for a specific task
func (r *ActionRepositoryImpl) ListByTask(ctx context.Context, taskID string) ([]*entity.Action, error) {
	return r.ListByScope(ctx, valueobject.ActionScopeTask, taskID)
}

// ListEnabled retrieves all enabled actions
func (r *ActionRepositoryImpl) ListEnabled(ctx context.Context) ([]*entity.Action, error) {
	allActions, err := r.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	enabled := make([]*entity.Action, 0)
	for _, action := range allActions {
		if action.Enabled() {
			enabled = append(enabled, action)
		}
	}

	return enabled, nil
}

// ListByTriggerType retrieves actions by trigger type
func (r *ActionRepositoryImpl) ListByTriggerType(ctx context.Context, triggerType entity.TriggerType) ([]*entity.Action, error) {
	allActions, err := r.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*entity.Action, 0)
	for _, action := range allActions {
		if action.Trigger().Type() == triggerType {
			filtered = append(filtered, action)
		}
	}

	return filtered, nil
}

// UpdateLastRun updates the last run time for an action
func (r *ActionRepositoryImpl) UpdateLastRun(ctx context.Context, id string) error {
	action, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	action.MarkAsRun()
	return r.Update(ctx, action)
}

// StorageAction represents an action in storage format
type StorageAction struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Scope       string                 `yaml:"scope"`
	ScopeID     string                 `yaml:"scope_id"`
	Enabled     bool                   `yaml:"enabled"`
	Trigger     config.TriggerConfig   `yaml:"trigger"`
	ActionType  config.ActionTypeConfig `yaml:"action_type"`
	Conditions  []config.ConditionConfig `yaml:"conditions,omitempty"`
	CreatedAt   string                 `yaml:"created_at"`
	ModifiedAt  string                 `yaml:"modified_at"`
	LastRun     *string                `yaml:"last_run,omitempty"`
}

// toStorageAction converts a domain action to storage format
func (r *ActionRepositoryImpl) toStorageAction(action *entity.Action) *StorageAction {
	// TODO: Implement proper conversion from domain entities to storage format
	// This is a placeholder that needs to be implemented based on the actual
	// trigger and action type implementations
	return &StorageAction{
		ID:          action.ID(),
		Name:        action.Name(),
		Description: action.Description(),
		Scope:       action.Scope().String(),
		ScopeID:     action.ScopeID(),
		Enabled:     action.Enabled(),
		CreatedAt:   action.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		ModifiedAt:  action.ModifiedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// fromStorageAction converts a storage action to domain entity
func (r *ActionRepositoryImpl) fromStorageAction(storageAction *StorageAction) (*entity.Action, error) {
	// TODO: Implement proper conversion from storage format to domain entities
	// This is a placeholder that needs to be implemented
	return nil, fmt.Errorf("not implemented")
}
