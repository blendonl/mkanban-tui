# mkanban

A powerful terminal-based Kanban board system with git workflow integration, featuring both an interactive TUI and comprehensive CLI.

## Architecture

This is a monorepo with a client-daemon architecture:

- **cmd/mkanban** - Terminal UI client that connects to the daemon
- **cmd/mkanbad** - Background daemon that manages board state and persistence
- **internal/daemon** - Unix socket IPC server with real-time update support
  - **protocol.go** - Request/response and notification protocol
  - **server.go** - Daemon server with subscription management
  - **client.go** - Client library for connecting to daemon
- **internal/model** - Shared data models (Board, Column, Task)
- **internal/storage** - File-based persistence layer
- **tui/** - TUI-specific components (view, update, styles)

The TUI client automatically starts the daemon if it's not already running and subscribes to real-time board updates.

## Building

### Prerequisites
- Go 1.24 or later
- Git (for git workflow features)
- Make (optional, for using Makefile)

### Quick Build
```bash
# Build both binaries
make build

# Or manually:
go build -o mkanban ./cmd/mkanban
go build -o mkanbad ./cmd/mkanbad
```

### Development
```bash
# Run tests
make test

# Format code
make fmt

# Run linters
make lint

# Generate wire DI
make wire

# Generate coverage report
make coverage
```

## Features

- ✅ **Multiple Boards** - Organize different projects or workflows
- ✅ **Interactive TUI** - Full-featured terminal user interface with daemon integration
- ✅ **Comprehensive CLI** - Complete command-line interface for all operations
- ✅ **Daemon Architecture** - Background daemon for multi-client support and real-time updates
- ✅ **Real-time Updates** - Live board updates across all connected TUI clients
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

### Arch Linux

#### From AUR (when published)
```bash
# Using yay
yay -S mkanban-tui

# Using paru
paru -S mkanban-tui
```

#### Build from source (Arch Linux)
```bash
# Clone the repository
git clone https://github.com/blendonl/mkanban-tui.git
cd mkanban-tui

# Build and install package
makepkg -sfi

# Or use the Makefile
make arch-install
```

### From Source (Other Linux Distributions)

#### Automated Installation
```bash
# Clone the repository
git clone https://github.com/blendonl/mkanban-tui.git
cd mkanban-tui

# Install system-wide (requires sudo)
sudo ./install.sh

# Or install to user directory
PREFIX=$HOME/.local ./install.sh
```

#### Using Make
```bash
# Build binaries
make build

# Install system-wide (requires sudo)
sudo make install

# Or install to custom prefix
PREFIX=$HOME/.local make install
```

#### Manual Build
```bash
# Build TUI client
go build -o mkanban ./cmd/mkanban

# Build daemon
go build -o mkanbad ./cmd/mkanbad

# Install to PATH
sudo mv mkanban /usr/local/bin/
sudo mv mkanbad /usr/local/bin/
```

### Enable Systemd Service (Optional but Recommended)

The daemon can be automatically started on login:

```bash
# Install systemd user service (if not already installed)
mkdir -p ~/.config/systemd/user
cp systemd/mkanbad.service ~/.config/systemd/user/

# Enable and start the service
systemctl --user daemon-reload
systemctl --user enable --now mkanbad.service

# Check status
systemctl --user status mkanbad.service
```

For system-wide daemon (multiple users):
```bash
# Install system service
sudo cp systemd/mkanbad@.service /usr/lib/systemd/system/

# Enable for specific user
sudo systemctl enable --now mkanbad@username.service
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
- `list_boards` - List all boards
- `create_board` - Create a new board
- `add_task` - Add a new task
- `move_task` - Move task between columns
- `update_task` - Update task details
- `delete_task` - Delete a task
- `add_column` - Add a new column
- `delete_column` - Remove a column
- `get_active_board` - Get the active board for current session
- `subscribe` - Subscribe to real-time board updates
- `ping` - Health check

**Real-time Updates:**
- Clients can subscribe to board changes via persistent connections
- The daemon broadcasts notifications when tasks are created, moved, updated, or deleted
- All connected TUI clients receive updates automatically

## Next Steps

- [x] Integrate TUI client with daemon
- [x] Implement real-time updates when daemon notifies changes
- [x] Add systemd service file for daemon auto-start
- [ ] Add task editing dialog in TUI
- [ ] Add column management in TUI
- [ ] Publish to AUR (Arch User Repository)
- [ ] Add configuration UI for daemon settings
- [ ] Add support for multiple simultaneous TUI instances

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Setup

1. Clone the repository
2. Install Go 1.24 or later
3. Run `make deps` to download dependencies
4. Run `make build` to build binaries
5. Run `make test` to run tests

### Project Structure

See the [Architecture](#architecture) section for an overview of the codebase structure.
