# Actions & Reminders Implementation

## Overview

A comprehensive action/reminder system for the mkanban-tui application that supports time-based scheduling, event triggers, automated task actions, and external integrations across global, board, column, and task scopes.

## Implementation Status

### ✅ Phase 1: Domain Layer (COMPLETE)

**Entities** (`internal/domain/entity/`)
- ✅ `Action` - Main action entity with ID, name, scope, trigger, action type, conditions
- ✅ `Trigger` interface with implementations:
  - `TimeTrigger` - Absolute time, relative to due date/creation, recurring (cron)
  - `EventTrigger` - Event-based triggers
- ✅ `ActionType` interface with implementations:
  - `NotificationAction` - Desktop notifications with template support
  - `ScriptAction` - Custom script execution with environment variables
  - `TaskMutationAction` - Update task fields (priority, status, tags, metadata)
  - `TaskMovementAction` - Move tasks between columns
  - `TaskCreationAction` - Create new tasks from templates
- ✅ `Condition` and `ConditionGroup` - Filtering conditions for actions
- ✅ `DomainEvent` and `EventBus` - Event system

**Value Objects** (`internal/domain/valueobject/`)
- ✅ `ActionScope` - Global, Board, Column, Task
- ✅ `EventType` - All domain events (task created, moved, status changed, etc.)
- ✅ `Schedule` - Time-based scheduling with 4 types

**Repository Interface** (`internal/domain/repository/`)
- ✅ `ActionRepository` - Complete CRUD and query interface

### ✅ Phase 2: Infrastructure Layer (COMPLETE)

**Configuration** (`internal/infrastructure/config/`)
- ✅ `ActionsConfig` - Main actions configuration with templates
- ✅ Action templates, triggers, schedules, and conditions config structures
- ✅ Default configuration with actions enabled

**Services** (`internal/infrastructure/external/`)
- ✅ `DesktopNotifier` - OS-specific implementations (Linux/macOS/Windows)
- ✅ `ScriptExecutor` - Secure script execution with validation

**Event Bus** (`internal/infrastructure/service/`)
- ✅ `EventBusImpl` - Thread-safe event publishing/subscription

**Storage** (`internal/infrastructure/persistence/filesystem/`)
- ✅ `ActionRepositoryImpl` - Filesystem-based YAML storage

### ✅ Phase 3: Application Layer (COMPLETE)

**Action Management Use Cases** (`internal/application/usecase/action/`)
- ✅ `CreateActionUseCase` - Create new actions
- ✅ `UpdateActionUseCase` - Update existing actions
- ✅ `DeleteActionUseCase` - Delete actions
- ✅ `GetActionUseCase` - Retrieve single action
- ✅ `ListActionsUseCase` - List actions with filters
- ✅ `EnableActionUseCase` - Enable actions
- ✅ `DisableActionUseCase` - Disable actions

**Action Execution Use Cases** (`internal/application/usecase/action/`)
- ✅ `EvaluateActionsUseCase` - Evaluate which actions should trigger
- ✅ `ExecuteActionUseCase` - Execute a single action
- ✅ `ProcessEventUseCase` - Process domain events and trigger actions

### ✅ Phase 4: Daemon Integration (COMPLETE)

**ActionManager** (`internal/daemon/`)
- ✅ `ActionManager` - Integrated into daemon server
  - Time-based scheduler (checks every 60 seconds by default)
  - Event listener subscribing to domain events
  - Action evaluation and execution
  - Starts/stops with daemon

**Dependency Injection** (`internal/di/`)
- ✅ All action services added to DI container
- ✅ Action repositories, use cases, and services wired
- ✅ TaskMutator service implementation
- ✅ EventBus, Notifier, ScriptRunner providers

**Daemon Server** (`internal/daemon/server.go`)
- ✅ ActionManager integrated into startup sequence
- ✅ Graceful shutdown handling
- ✅ Action protocol endpoints added

### ✅ Phase 4.5: Configuration & Templates (COMPLETE)

**Action Manager** (`internal/daemon/`)
- ✅ `ActionManager` - Main service managing actions
  - Time-based scheduler (checks every 60 seconds by default)
  - Event listener subscribing to domain events
  - Action evaluation and execution

**Protocol** (`internal/daemon/protocol.go`)
- ✅ Action request types:
  - `create_action`, `update_action`, `delete_action`
  - `get_action`, `list_actions`
  - `enable_action`, `disable_action`
- ✅ Request/response payload structures

**Example Templates** (`config.yml`)
- ✅ Due date reminder template included
- ✅ Template shows all configuration options
- ✅ Ready-to-use out of the box

### ⏳ Phase 5: TUI Interface (PENDING)

**Views Needed:**
- [ ] Action list view (global + current context)
- [ ] Action creation/edit form
- [ ] Action template browser
- [ ] Enable/disable toggle
- [ ] Integration with board/column/task views

### ⏳ Phase 6: Testing (PENDING)

**Tests Needed:**
- [ ] Domain entity unit tests
- [ ] Trigger evaluation tests
- [ ] Condition matching tests
- [ ] Use case tests
- [ ] Integration tests

## Configuration Example

```yaml
actions:
  enabled: true
  check_interval: 60  # Check time-based actions every 60 seconds
  notifications_enabled: true
  scripts_enabled: true
  scripts_dir: ~/.config/mkanban/scripts
  templates:
    - id: "overdue-reminder"
      name: "Overdue Task Reminder"
      description: "Notify when a task is overdue"
      trigger:
        type: "event"
        event: "task.overdue"
      action_type:
        type: "notification"
        title: "Task Overdue"
        message: "Task {{task.title}} is overdue!"
      conditions:
        - field: "priority"
          operator: "in"
          value: ["high", "medium"]
```

## Action Types

### 1. Notification Action
Sends desktop notifications with template support.

**Configuration:**
```yaml
action_type:
  type: "notification"
  title: "Task Due Soon"
  message: "{{task.title}} is due in 1 hour"
```

### 2. Script Action
Executes custom scripts with environment variables.

**Configuration:**
```yaml
action_type:
  type: "script"
  script_path: "notify_slack.sh"
  script_env:
    SLACK_WEBHOOK: "https://..."
```

**Environment Variables Available:**
- `TASK_ID`, `TASK_TITLE`, `TASK_PRIORITY`, `TASK_STATUS`
- `BOARD_ID`, `BOARD_NAME`
- `COLUMN_NAME`

### 3. Task Mutation Action
Updates task fields automatically.

**Configuration:**
```yaml
action_type:
  type: "task_mutation"
  update_priority: "high"
  add_tags: ["urgent"]
  set_metadata:
    auto_escalated: "true"
```

### 4. Task Movement Action
Moves tasks between columns.

**Configuration:**
```yaml
action_type:
  type: "task_movement"
  target_column: "in-progress"
```

### 5. Task Creation Action
Creates new tasks automatically.

**Configuration:**
```yaml
action_type:
  type: "task_creation"
  task_title: "Follow-up: {{task.title}}"
  task_description: "Created as follow-up"
  task_column: "backlog"
  task_priority: "medium"
```

## Triggers

### Time-Based Triggers

**Absolute Time:**
```yaml
trigger:
  type: "time"
  schedule:
    type: "absolute"
    time: "2025-10-25T14:00:00Z"
```

**Relative to Due Date:**
```yaml
trigger:
  type: "time"
  schedule:
    type: "relative_due_date"
    offset: "1h"  # 1 hour before due date
```

**Relative to Creation:**
```yaml
trigger:
  type: "time"
  schedule:
    type: "relative_creation"
    offset: "3d"  # 3 days after creation
```

**Recurring (Cron):**
```yaml
trigger:
  type: "time"
  schedule:
    type: "recurring"
    cron: "0 9 * * *"  # Every day at 9am
```

### Event-Based Triggers

```yaml
trigger:
  type: "event"
  event: "task.moved"
```

**Available Events:**
- `task.created`, `task.updated`, `task.deleted`, `task.moved`
- `task.status_changed`, `task.priority_changed`
- `task.due_date_set`, `task.due_date_changed`
- `task.completed`, `task.overdue`
- `column.created`, `column.deleted`, `column.wip_reached`

## Conditions

Filter when actions execute with conditions:

```yaml
conditions:
  - field: "priority"
    operator: "eq"
    value: "high"
  - field: "column"
    operator: "in"
    value: ["in-progress", "review"]
  - field: "tags"
    operator: "contains"
    value: "urgent"
```

**Operators:**
- `eq`, `ne` - Equals, not equals
- `contains`, `not_contains` - Contains string/tag
- `in`, `not_in` - In list
- `gt`, `lt` - Greater than, less than

**Fields:**
- `priority`, `status`, `column`, `tags`
- `has_due_date`, `is_overdue`
- Any metadata key

## Action Scopes

Actions can be defined at different scopes:

1. **Global** - Applies to all boards
2. **Board** - Applies to specific board
3. **Column** - Applies to specific column
4. **Task** - Applies to specific task

Actions are inherited: global → board → column → task

## Usage Examples

### Example 1: Due Date Reminder
```yaml
- id: "due-reminder"
  name: "Due Date Reminder"
  trigger:
    type: "time"
    schedule:
      type: "relative_due_date"
      offset: "1d"
  action_type:
    type: "notification"
    title: "Task Due Tomorrow"
    message: "{{task.title}} is due tomorrow"
```

### Example 2: Auto-escalate Stale Tasks
```yaml
- id: "auto-escalate"
  name: "Auto-escalate Stale Tasks"
  trigger:
    type: "time"
    schedule:
      type: "relative_creation"
      offset: "7d"
  action_type:
    type: "task_mutation"
    update_priority: "high"
    add_tags: ["stale"]
  conditions:
    - field: "status"
      operator: "eq"
      value: "todo"
```

### Example 3: Move Completed Tasks
```yaml
- id: "move-done"
  name: "Move Completed Tasks"
  trigger:
    type: "event"
    event: "task.completed"
  action_type:
    type: "task_movement"
    target_column: "done"
```

### Example 4: Notify External System
```yaml
- id: "notify-slack"
  name: "Notify Slack on High Priority"
  trigger:
    type: "event"
    event: "task.created"
  action_type:
    type: "script"
    script_path: "notify_slack.sh"
  conditions:
    - field: "priority"
      operator: "eq"
      value: "high"
```

## Next Steps

1. **Complete TUI Interface** - Add UI for managing actions
2. **Add Tests** - Comprehensive test coverage
3. **Template Engine** - Better variable interpolation for notifications
4. **Cron Parser** - Implement recurring schedule evaluation
5. **Action History** - Log of executed actions
6. **Action Metrics** - Track action performance

## Files Created

- Domain: `action.go`, `action_type.go`, `trigger.go`, `condition.go`, `domain_event.go`
- Value Objects: `action_scope.go`, `event_type.go`, `schedule.go`
- Repository: `action_repository.go`, `action_repository_impl.go`
- Services: `desktop_notifier.go`, `script_executor.go`, `event_bus.go`
- Use Cases: `create_action.go`, `update_action.go`, `delete_action.go`, `get_action.go`, `list_actions.go`, `enable_action.go`, `evaluate_actions.go`, `execute_action.go`, `process_event.go`
- Daemon: `action_manager.go`, protocol updates
- Config: Actions configuration added to `config.go`
