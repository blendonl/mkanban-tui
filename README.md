# mkanban

A powerful terminal-based Kanban board system with git workflow integration, featuring both an interactive TUI and comprehensive CLI.

## Architecture

This is a monorepo containing:

- **cmd/mkanban** - Terminal UI client for viewing and managing the Kanban board
- **cmd/mkanbad** - Background daemon that manages board state and persistence
- **internal/model** - Shared data models (Board, Column, Task)
- **internal/storage** - File-based persistence layer
- **internal/daemon** - Unix socket IPC server
- **tui/** - TUI-specific components (view, update, styles)

## Building

Build both binaries:

```bash
# Build TUI client
go build -o mkanban ./cmd/mkanban

# Build daemon
go build -o mkanbad ./cmd/mkanbad
```

## Features

- ✅ **Multiple Boards** - Organize different projects or workflows
- ✅ **Interactive TUI** - Full-featured terminal user interface
- ✅ **Comprehensive CLI** - Complete command-line interface for all operations
- ✅ **Git Integration** - Checkout branches for tasks automatically
- ✅ **Task Management** - Priorities, tags, due dates, descriptions
- ✅ **Automated Actions** - Time-based and event-based task automation
- ✅ **Tmux Integration** - Session-aware board switching
- ✅ **Multiple Output Formats** - Text, JSON, YAML for scripting
- ✅ **Shell Completion** - Bash, Zsh, Fish, PowerShell support

## Quick Start

### Interactive TUI (Default)

```bash
# Launch interactive TUI
mkanban

# Launch TUI for specific board
mkanban --board-id my-project
```

### CLI Commands

```bash
# List all boards
mkanban board list

# Create a new task
mkanban task create --title "Fix login bug" --priority high

# List tasks in a column
mkanban task list --column "In Progress"

# Move task to next column
mkanban task advance TASK-123

# Checkout git branch for task
mkanban task checkout TASK-123

# Get help
mkanban --help
mkanban task --help
```

## Installation

### From Source

```bash
go build -o mkanban ./cmd/mkanban
go build -o mkanbad ./cmd/mkanbad

# Install to PATH
sudo mv mkanban /usr/local/bin/
sudo mv mkanbad /usr/local/bin/
```

### Shell Completion

```bash
# Bash
mkanban completion bash > /etc/bash_completion.d/mkanban

# Zsh
mkanban completion zsh > "${fpath[1]}/_mkanban"

# Fish
mkanban completion fish > ~/.config/fish/completions/mkanban.fish
```

## CLI Reference

### Global Flags

Available for all commands:

```bash
--board-id, -b <id>     Board to operate on (default: active board)
--output, -o <format>   Output format: text, json, yaml (default: text)
--config, -c <path>     Config file path
--quiet, -q             Suppress non-essential output
--help, -h              Show help
--version, -v           Show version
```

### Board Commands

Manage multiple kanban boards:

```bash
# List all boards
mkanban board list
mkanban board list --output json

# Get board details
mkanban board get my-project

# Create a new board
mkanban board create my-project \
  --name "My Project" \
  --description "Project tasks" \
  --columns "Todo,In Progress,Review,Done"

# Show current active board
mkanban board current

# Switch active board
mkanban board switch my-project

# Delete a board
mkanban board delete my-project
```

### Column Commands

Manage columns within boards:

```bash
# List columns
mkanban column list
mkanban column list --board-id my-project

# Get column details
mkanban column get "In Progress"

# Create a column
mkanban column create "Code Review" --position 3 --wip-limit 5

# Update column
mkanban column update "In Progress" --wip-limit 5

# Reorder columns
mkanban column reorder "Backlog,Todo,In Progress,Review,Done"

# Delete column
mkanban column delete "Archived" --move-tasks-to "Done"
```

### Task Commands

Complete task management with all TUI features:

```bash
# List tasks
mkanban task list
mkanban task list --column "Todo"
mkanban task list --priority high
mkanban task list --overdue
mkanban task list --tag urgent
mkanban task list --output json

# Get task details
mkanban task get TASK-123
mkanban task get TASK-123 --output markdown

# Create task
mkanban task create \
  --title "Implement feature X" \
  --description "Add new feature" \
  --priority high \
  --column "Todo" \
  --tags "backend,api" \
  --due "2025-12-31"

# Create task with editor
mkanban task create --title "Write docs" --edit

# Update task
mkanban task update TASK-123 \
  --priority critical \
  --add-tag urgent \
  --due "2025-11-30"

# Edit task description
mkanban task update TASK-123 --edit

# Move task to specific column
mkanban task move TASK-123 "In Progress"

# Move task to next column (like TUI 'm' key)
mkanban task advance TASK-123

# Move task to previous column
mkanban task retreat TASK-123

# Delete task
mkanban task delete TASK-123

# Checkout git branch for task
mkanban task checkout TASK-123
mkanban task checkout TASK-123 --branch-format "feature/{short-id}-{slug}"

# Show task with context
mkanban task show TASK-123 --context 5
```

### Config Commands

Manage configuration:

```bash
# Show configuration
mkanban config show
mkanban config show --output yaml

# Edit config in editor
mkanban config edit

# Show config file path
mkanban config path

# Reset to defaults
mkanban config reset
```

### Other Commands

```bash
# Migrate data formats
mkanban migrate

# Generate shell completions
mkanban completion bash
mkanban completion zsh
mkanban completion fish
```

## Output Formats

### Text (Default)

Human-readable table output:

```
TODO
  ⚫ TASK-001 Fix login bug           📅 due tomorrow
  ⚪ TASK-002 Update documentation

IN PROGRESS
  ⚫ TASK-003 Implement API           📅 overdue 2 days
```

### JSON

For scripting and automation:

```bash
mkanban task list --output json | jq '.[] | select(.priority == "high")'
```

### YAML

For configuration and readability:

```bash
mkanban board get my-project --output yaml > board-backup.yml
```

### Path Format

For integration with other tools:

```bash
mkanban task list --output path
# Output: boards/my-project/todo/TASK-001-fix-bug :: Fix login bug
```

## Git Workflow Integration

### Branch Checkout

Automatically checkout git branches for tasks:

```bash
# Create task and checkout branch
TASK_ID=$(mkanban task create --title "Add dark mode" --output json | jq -r '.short_id')
mkanban task checkout $TASK_ID

# Custom branch format
mkanban task checkout TASK-123 --branch-format "feature/{short-id}-{slug}"
```

Available placeholders:
- `{id}` - Full task ID (e.g., TASK-123-add-dark-mode)
- `{short-id}` - Short ID (e.g., TASK-123)
- `{slug}` - Title slug (e.g., add-dark-mode)

## TUI (Interactive Mode)

### Keybindings

- **Navigation**
  - `←/h` - Move to left column
  - `→/l` - Move to right column
  - `↑/k` - Move to task above
  - `↓/j` - Move to task below

- **Actions**
  - `a` - Add new task
  - `d` - Delete selected task
  - `m/Enter` - Move task to next column
  - `q/Ctrl+C` - Quit

## Project Structure

```
mkanban/
├── cmd/
│   ├── mkanban/         # TUI client
│   │   └── main.go
│   └── mkanbad/         # Daemon
│       └── main.go
├── internal/
│   ├── model/          # Data models
│   │   ├── board.go
│   │   ├── column.go
│   │   └── task.go
│   ├── storage/        # Persistence layer
│   │   └── storage.go
│   └── daemon/         # IPC server
│       ├── protocol.go
│       └── server.go
├── tui/                # TUI components
│   ├── model.go
│   ├── view.go
│   ├── update.go
│   ├── keymap.go
│   └── style/
│       └── tui-style.go
├── go.mod
└── README.md
```

## Communication Protocol

The daemon uses Unix sockets with JSON-based request/response protocol:

**Request Types:**
- `get_board` - Retrieve current board state
- `add_task` - Add a new task
- `move_task` - Move task between columns
- `update_task` - Update task details
- `delete_task` - Delete a task
- `add_column` - Add a new column
- `delete_column` - Remove a column

## Next Steps

- [ ] Integrate TUI client with daemon (currently runs standalone)
- [ ] Add task editing dialog in TUI
- [ ] Add column management in TUI
- [ ] Implement real-time updates when daemon notifies changes
- [ ] Add systemd service file for daemon
- [ ] Add task descriptions and metadata
- [ ] Add task priorities and tags
