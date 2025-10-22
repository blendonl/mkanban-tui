# mkanban

A terminal-based Kanban board application with daemon and TUI components.

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

## Running

### Standalone TUI (no daemon)

```bash
./mkanban
```

The TUI will run with an in-memory board (changes not persisted).

### With Daemon

1. Start the daemon in the background:

```bash
./mkanbad &
```

The daemon will:
- Listen on Unix socket at `~/.local/share/mkanban/mkanbad.sock`
- Persist board state to `~/.local/share/mkanban/board.json`

2. Run the TUI client (future enhancement will connect to daemon):

```bash
./mkanban
```

## Keybindings

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
