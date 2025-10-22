# mkanban Architecture

## Overview

This project implements a terminal-based Kanban board using **Clean Architecture** principles, ensuring maintainability, testability, and extensibility.

## Architecture Layers

### 1. Domain Layer (`internal/domain/`)

The core business logic layer, containing:

#### Entities (`entity/`)
- **Board**: Aggregate root managing columns and tasks
- **Column**: Contains tasks with WIP limits
- **Task**: Represents work items

#### Value Objects (`valueobject/`)
- **TaskID**: Immutable ID in format `PREFIX-NUM-slug` (e.g., `REC-001-setup-database`)
- **Priority**: Enumeration (none, low, medium, high, critical)
- **Status**: Task status (todo, in_progress, blocked, in_review, done)
- **Color**: Hex color value for visual customization

#### Repository Interfaces (`repository/`)
- **BoardRepository**: Interface for board persistence

#### Domain Services (`service/`)
- **BoardService**: Complex business operations
- **ValidationService**: Business rule validation

#### Design Patterns Used
- **Aggregate Pattern**: Board is the aggregate root
- **Value Objects**: Immutable types preventing invalid states
- **Repository Pattern**: Abstraction over storage
- **Domain Services**: Complex business logic not belonging to entities

### 2. Application Layer (`internal/application/`)

Orchestrates domain operations and handles use cases:

#### DTOs (`dto/`)
Data transfer objects for external communication:
- `BoardDTO`, `ColumnDTO`, `TaskDTO`
- Request/Response objects

#### Use Cases (`usecase/`)
Single-responsibility operations:
- **board/**: `CreateBoardUseCase`, `GetBoardUseCase`, `ListBoardsUseCase`
- **column/**: `CreateColumnUseCase`
- **task/**: `CreateTaskUseCase`, `MoveTaskUseCase`, `UpdateTaskUseCase`

#### Design Patterns Used
- **Use Case Pattern**: One class per operation
- **DTO Pattern**: Decouples layers
- **Command Pattern**: Request objects encapsulate operations

### 3. Infrastructure Layer (`internal/infrastructure/`)

Handles external concerns:

#### Persistence (`persistence/`)
- **filesystem/**: File-based storage implementation
  - `BoardRepositoryImpl`: Implements `BoardRepository`
  - `PathBuilder`: Constructs filesystem paths
- **mapper/**: Converts between entities and storage format

#### Serialization (`serialization/`)
- **FrontmatterParser**: Parses YAML frontmatter in markdown files

#### Configuration (`config/`)
- **Config**: Application configuration
- **Loader**: Loads/saves config from filesystem

#### Design Patterns Used
- **Repository Implementation**: Concrete implementation of repository interface
- **Mapper Pattern**: Entity ↔ Storage conversion
- **Factory Pattern**: Path construction

### 4. Presentation Layer

#### TUI (`tui/`)
Terminal user interface using Bubble Tea

#### Daemon (`internal/daemon/`)
Background service for persistence and IPC

### 5. Dependency Injection (`internal/di/`)

Uses **Google Wire** for compile-time dependency injection:
- `wire.go`: Dependency definitions
- `wire_gen.go`: Generated wiring code

## File Storage Structure

```
~/.local/share/mkanban/
├── config.json              # Application configuration
└── boards/
    └── project-alpha/       # Board folder (name = board ID)
        ├── board.md         # Board metadata (YAML frontmatter)
        ├── todo/            # Column folder
        │   ├── column.md    # Column metadata
        │   ├── PRO-001-setup-db/     # Task folder (TaskID format)
        │   │   └── task.md           # Task metadata + content
        │   └── PRO-002-api-design/
        │       └── task.md
        ├── in-progress/
        │   ├── column.md
        │   └── PRO-003-auth/
        │       └── task.md
        └── done/
            ├── column.md
            └── PRO-004-init/
                └── task.md
```

### Metadata Format (YAML Frontmatter)

**board.md:**
```yaml
---
id: project-alpha
prefix: PRO
created: 2025-10-22T10:30:00Z
modified: 2025-10-22T14:45:00Z
description: Main project board
next_task_num: 5
---
```

**column.md:**
```yaml
---
description: Tasks in progress
order: 2
wip_limit: 5
color: "#3498db"
---
```

**task.md:**
```yaml
---
id: PRO-001
title: Setup Database
created: 2025-10-22T10:30:00Z
modified: 2025-10-22T14:45:00Z
due_date: 2025-10-30T23:59:59Z
priority: high
status: in_progress
tags: [backend, database]
---
Setup PostgreSQL database with proper schema and migrations.

Additional notes and description go here...
```

## Key Business Rules

1. **Task ID Format**: `{BOARD_PREFIX}-{NUMBER}-{SLUG}`
   - Board prefix: First 3 letters of board name (uppercase)
   - Number: Auto-incrementing per board
   - Slug: Sanitized task title

2. **WIP Limits**: Enforced at column level (0 = unlimited)

3. **Name Uniqueness**:
   - Board names must be unique across all boards
   - Column names must be unique within a board
   - Task IDs are globally unique

4. **Required Fields**:
   - Board: name
   - Column: name
   - Task: title, ID

5. **Validation**:
   - Names: 3-50 chars for boards, 1-30 for columns, 1-100 for tasks
   - Colors: Hex format (#RGB or #RRGGBB)
   - Due dates: Cannot be in the past

## SOLID Principles

✅ **Single Responsibility**: Each class/module has one reason to change
✅ **Open/Closed**: Extensible without modification (interfaces, value objects)
✅ **Liskov Substitution**: Repository implementations are interchangeable
✅ **Interface Segregation**: Small, focused interfaces
✅ **Dependency Inversion**: High-level modules depend on abstractions

## Testing Strategy

### Unit Tests
- Domain entities and value objects
- Domain services
- Use cases (with mocked repositories)

### Integration Tests
- Repository implementations
- Filesystem operations
- End-to-end use case flows

### Test Coverage Areas
1. `internal/domain/entity/` - Entity behavior
2. `internal/domain/valueobject/` - Value object validation
3. `internal/domain/service/` - Business rules
4. `internal/infrastructure/persistence/` - Storage operations

## Dependencies

### Core
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/google/wire` - Dependency injection

### TUI
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling

## Future Extensions

### Easy to Add
1. **New Storage Backend**: Implement `BoardRepository` interface
   - Example: PostgreSQL, SQLite, MongoDB
2. **New Entity Types**: Add to domain layer
   - Example: Projects, Sprints, Comments
3. **New Use Cases**: Add to application layer
   - Example: Search, Reports, Export
4. **Event Sourcing**: Add domain events to entities

### Architectural Benefits
- **Testability**: Mock interfaces, test in isolation
- **Maintainability**: Clear separation of concerns
- **Extensibility**: Add features without touching core
- **Portability**: Swap implementations easily

## Getting Started

### Build
```bash
# Generate Wire code (when dependencies change)
go run github.com/google/wire/cmd/wire gen ./internal/di

# Build all
go build ./...

# Build TUI
go build -o mkanban ./cmd/mkanban

# Build daemon
go build -o mkanbad ./cmd/mkanbad
```

### Usage
```bash
# Run TUI
./mkanban

# Run daemon
./mkanbad
```

### Configuration
Config file: `~/.local/share/mkanban/config.json`

```json
{
  "boards_path": "/home/user/.local/share/mkanban/boards",
  "data_path": "/home/user/.local/share/mkanban"
}
```

## Design Principles Applied

1. **Domain-Driven Design (DDD)**
   - Ubiquitous language
   - Bounded contexts
   - Aggregates

2. **Clean Architecture**
   - Dependency rule (inward dependencies only)
   - Layers of abstraction
   - Framework independence

3. **CQRS (Command Query Responsibility Segregation)**
   - Use cases separate reads and writes
   - DTOs for queries
   - Commands modify state

4. **Immutability**
   - Value objects are immutable
   - Entities expose behavior, not data

## Code Organization

```
mkanban/
├── cmd/                     # Entry points
├── internal/
│   ├── domain/             # ← Pure business logic (no dependencies)
│   ├── application/        # ← Use cases (depends on domain)
│   ├── infrastructure/     # ← External concerns (depends on domain)
│   ├── daemon/             # ← Daemon-specific code
│   └── di/                 # ← Dependency injection
├── tui/                    # TUI presentation
└── pkg/                    # Shared utilities
```

### Dependency Flow
```
Presentation → Application → Domain
                ↓
          Infrastructure
```

All layers depend on Domain (Dependency Inversion).
