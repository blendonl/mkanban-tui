package daemon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mkanban/internal/application/usecase/action"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/config"
)

// ActionManager manages action evaluation and execution
type ActionManager struct {
	config              *config.Config
	evaluateUseCase     *action.EvaluateActionsUseCase
	executeUseCase      *action.ExecuteActionUseCase
	processEventUseCase *action.ProcessEventUseCase
	actionRepo          repository.ActionRepository
	eventBus            entity.EventBus

	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

// NewActionManager creates a new ActionManager
func NewActionManager(
	cfg *config.Config,
	evaluateUseCase *action.EvaluateActionsUseCase,
	executeUseCase *action.ExecuteActionUseCase,
	processEventUseCase *action.ProcessEventUseCase,
	actionRepo repository.ActionRepository,
	eventBus entity.EventBus,
) *ActionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ActionManager{
		config:              cfg,
		evaluateUseCase:     evaluateUseCase,
		executeUseCase:      executeUseCase,
		processEventUseCase: processEventUseCase,
		actionRepo:          actionRepo,
		eventBus:            eventBus,
		ctx:                 ctx,
		cancelFunc:          cancel,
	}
}

// Start starts the action manager
func (m *ActionManager) Start() error {
	if !m.config.Actions.Enabled {
		fmt.Println("Actions are disabled in configuration")
		return nil
	}

	fmt.Println("Starting ActionManager...")

	// Start time-based scheduler
	m.wg.Add(1)
	go m.runScheduler()

	// Subscribe to all domain events
	m.subscribeToEvents()

	fmt.Println("ActionManager started successfully")
	return nil
}

// Stop stops the action manager
func (m *ActionManager) Stop() error {
	fmt.Println("Stopping ActionManager...")

	m.cancelFunc()
	m.wg.Wait()

	fmt.Println("ActionManager stopped")
	return nil
}

// runScheduler runs the time-based action scheduler
func (m *ActionManager) runScheduler() {
	defer m.wg.Done()

	checkInterval := time.Duration(m.config.Actions.CheckInterval) * time.Second
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	fmt.Printf("Time-based scheduler started (checking every %v)\n", checkInterval)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkTimeBasedActions()
		}
	}
}

// checkTimeBasedActions checks and executes time-based actions
func (m *ActionManager) checkTimeBasedActions() {
	ctx := context.Background()

	// Get all time-based actions
	timeTriggerType := entity.TriggerTypeTime
	actions, err := m.actionRepo.ListByTriggerType(ctx, timeTriggerType)
	if err != nil {
		fmt.Printf("Failed to list time-based actions: %v\n", err)
		return
	}

	if len(actions) == 0 {
		return
	}

	fmt.Printf("Checking %d time-based actions...\n", len(actions))

	// Evaluate each action
	for _, act := range actions {
		if !act.Enabled() {
			continue
		}

		// Build evaluation context
		evalCtx := action.EvaluationContext{
			CurrentTime: time.Now(),
			// TODO: For time-based actions, we might need to iterate over all tasks
			// or have a way to identify which tasks to check
		}

		// For now, we'll evaluate without specific task context
		// This works for global time-based actions
		results, err := m.evaluateUseCase.Execute(ctx, evalCtx)
		if err != nil {
			fmt.Printf("Failed to evaluate action %s: %v\n", act.Name(), err)
			continue
		}

		// Execute triggered actions
		for _, result := range results {
			execReq := action.ExecutionRequest{
				Action:         result.Action,
				TriggerContext: result.Context,
			}

			if err := m.executeUseCase.Execute(ctx, execReq); err != nil {
				fmt.Printf("Failed to execute action %s: %v\n", result.Action.Name(), err)
			} else {
				fmt.Printf("Executed action: %s\n", result.Action.Name())
			}
		}
	}
}

// subscribeToEvents subscribes to all domain events
func (m *ActionManager) subscribeToEvents() {
	// Create event handler that processes events through the ProcessEventUseCase
	handler := func(event *entity.DomainEvent) {
		ctx := context.Background()

		if err := m.processEventUseCase.Execute(ctx, event); err != nil {
			fmt.Printf("Failed to process event %s: %v\n", event.Type, err)
		}
	}

	// Subscribe to all event types that we care about
	eventTypes := []valueobject.EventType{
		valueobject.EventTaskCreated,
		valueobject.EventTaskUpdated,
		valueobject.EventTaskDeleted,
		valueobject.EventTaskMoved,
	}

	for _, eventType := range eventTypes {
		// TODO: Subscribe to specific event types once we have proper EventType values
		_ = eventType
	}

	// For now, subscribe to all events
	// m.eventBus.SubscribeAll(handler)

	// Subscribe to specific event types
	m.eventBus.Subscribe("task.created", handler)
	m.eventBus.Subscribe("task.updated", handler)
	m.eventBus.Subscribe("task.deleted", handler)
	m.eventBus.Subscribe("task.moved", handler)
	m.eventBus.Subscribe("task.status_changed", handler)
	m.eventBus.Subscribe("task.priority_changed", handler)
	m.eventBus.Subscribe("task.due_date_set", handler)
	m.eventBus.Subscribe("task.due_date_changed", handler)
	m.eventBus.Subscribe("task.completed", handler)
	m.eventBus.Subscribe("column.created", handler)
	m.eventBus.Subscribe("column.deleted", handler)
	m.eventBus.Subscribe("column.wip_reached", handler)

	fmt.Println("Subscribed to domain events")
}

// PublishEvent is a helper to publish events (for testing and manual triggers)
func (m *ActionManager) PublishEvent(event *entity.DomainEvent) {
	m.eventBus.Publish(event)
}
