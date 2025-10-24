package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"mkanban/internal/application/dto"
	"mkanban/internal/daemon"
	"mkanban/internal/di"
	"mkanban/internal/infrastructure/config"
	"mkanban/tui"
	"mkanban/tui/style"
)

func main() {
	// Load configuration
	loader, err := config.NewLoader()
	if err != nil {
		log.Fatalf("Failed to create config loader: %v", err)
	}

	cfg, err := loader.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize styles and keybindings from config
	style.InitStyles(cfg)
	tui.InitKeybindings(cfg)

	// Initialize DI container
	container, err := di.InitializeContainer()
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}

	ctx := context.Background()

	// Check for CLI commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			if len(os.Args) > 2 && os.Args[2] == "task" {
				handleListTask(ctx, container, cfg, os.Args[3:])
				return
			}
		case "new":
			if len(os.Args) > 2 && os.Args[2] == "todo" {
				handleNewTodo(ctx, container, cfg, os.Args[3:])
				return
			}
		case "checkout":
			handleCheckout(ctx, container, cfg, os.Args[2:])
			return
		case "migrate":
			handleMigrate(ctx, container)
			return
		}
	}

	// Try to get active board from daemon if running in tmux
	var boardID string
	if isRunningInTmux() {
		activeBoardID, err := getActiveBoardFromDaemon(cfg)
		if err == nil && activeBoardID != "" {
			// Verify the board exists
			_, err := container.GetBoardUseCase.Execute(ctx, activeBoardID)
			if err == nil {
				boardID = activeBoardID
				fmt.Printf("Using active tmux session board\n")
			}
		}
	}

	// If no active board was found, use existing logic
	if boardID == "" {
		// Get list of boards
		boards, err := container.ListBoardsUseCase.Execute(ctx)
		if err != nil {
			log.Fatalf("Failed to list boards: %v", err)
		}

		// If no boards exist, create a default one
		if len(boards) == 0 {
			fmt.Println("No boards found. Creating default board...")

			// Create default board
			board, err := container.CreateBoardUseCase.Execute(ctx, struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}{
				Name:        "My Board",
				Description: "Default kanban board",
			})
			if err != nil {
				log.Fatalf("Failed to create default board: %v", err)
			}

			boardID = board.ID

			// Add default columns
			columns := []struct {
				name        string
				description string
				order       int
			}{
				{"Todo", "Tasks to be done", 1},
				{"In Progress", "Currently working on", 2},
				{"Done", "Completed tasks", 3},
			}

			for _, col := range columns {
				_, err := container.CreateColumnUseCase.Execute(ctx, boardID, struct {
					Name        string  `json:"name"`
					Description string  `json:"description"`
					Order       int     `json:"order"`
					WIPLimit    int     `json:"wip_limit"`
					Color       *string `json:"color,omitempty"`
				}{
					Name:        col.name,
					Description: col.description,
					Order:       col.order,
					WIPLimit:    0, // Unlimited
					Color:       nil,
				})
				if err != nil {
					log.Fatalf("Failed to create column %s: %v", col.name, err)
				}
			}

			fmt.Printf("Created board: %s\n", board.Name)
		} else {
			// Use the first board
			boardID = boards[0].ID
			fmt.Printf("Using board: %s\n", boards[0].Name)
		}
	}

	// Load the full board
	boardDTO, err := container.GetBoardUseCase.Execute(ctx, boardID)
	if err != nil {
		log.Fatalf("Failed to load board: %v", err)
	}

	// Create TUI model
	m := tui.NewModel(boardDTO, container)

	// Start the program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

// isRunningInTmux checks if the program is running inside a tmux session
func isRunningInTmux() bool {
	return os.Getenv("TMUX") != ""
}

// getActiveBoardFromDaemon requests the active board ID from the daemon
func getActiveBoardFromDaemon(cfg *config.Config) (string, error) {
	// Get socket path
	socketPath := daemon.GetSocketPath(cfg)

	// Connect to daemon
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	// Send request
	req := daemon.Request{
		Type: daemon.RequestGetActiveBoard,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	var resp daemon.Response
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("daemon error: %s", resp.Error)
	}

	// Extract board ID from response
	if resp.Data == nil {
		return "", nil
	}

	// The response data is a map with board_id key
	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	boardID, ok := dataMap["board_id"].(string)
	if !ok {
		return "", nil
	}

	return boardID, nil
}

// handleListTask handles the "list task" command
func handleListTask(ctx context.Context, container *di.Container, cfg *config.Config, args []string) {
	// Determine which board to use
	var boardID string

	// If a board ID is provided as argument, use it
	if len(args) > 0 {
		boardID = args[0]
	} else {
		// Try to get active board from daemon if running in tmux
		if isRunningInTmux() {
			activeBoardID, err := getActiveBoardFromDaemon(cfg)
			if err == nil && activeBoardID != "" {
				// Verify the board exists
				_, err := container.GetBoardUseCase.Execute(ctx, activeBoardID)
				if err == nil {
					boardID = activeBoardID
				}
			}
		}

		// If no active board was found, use the first board
		if boardID == "" {
			boards, err := container.ListBoardsUseCase.Execute(ctx)
			if err != nil {
				log.Fatalf("Failed to list boards: %v", err)
			}

			if len(boards) == 0 {
				log.Fatalf("No boards found. Create a board first by running mkanban without arguments.")
			}

			boardID = boards[0].ID
		}
	}

	// Execute the list tasks use case
	tasks, err := container.ListTasksUseCase.Execute(ctx, boardID)
	if err != nil {
		log.Fatalf("Failed to list tasks: %v", err)
	}

	// Output tasks in the format: path :: title
	for _, task := range tasks {
		fmt.Printf("%s :: %s\n", task.FilePath, task.Title)
	}
}

// handleNewTodo handles the "new todo" command
func handleNewTodo(ctx context.Context, container *di.Container, cfg *config.Config, args []string) {
	// Determine which board to use
	var boardID string

	// If a board ID is provided as argument, use it
	if len(args) > 0 {
		boardID = args[0]
	} else {
		// Try to get active board from daemon if running in tmux
		if isRunningInTmux() {
			activeBoardID, err := getActiveBoardFromDaemon(cfg)
			if err == nil && activeBoardID != "" {
				// Verify the board exists
				_, err := container.GetBoardUseCase.Execute(ctx, activeBoardID)
				if err == nil {
					boardID = activeBoardID
				}
			}
		}

		// If no active board was found, use the first board
		if boardID == "" {
			boards, err := container.ListBoardsUseCase.Execute(ctx)
			if err != nil {
				log.Fatalf("Failed to list boards: %v", err)
			}

			if len(boards) == 0 {
				log.Fatalf("No boards found. Create a board first by running mkanban without arguments.")
			}

			boardID = boards[0].ID
		}
	}

	// Get the board to verify it has a "Todo" column
	board, err := container.GetBoardUseCase.Execute(ctx, boardID)
	if err != nil {
		log.Fatalf("Failed to get board: %v", err)
	}

	// Check if "Todo" column exists (case-insensitive, handles variations)
	var todoColumnName string
	for _, column := range board.Columns {
		if isTodoColumn(column.Name) {
			todoColumnName = column.Name
			break
		}
	}

	if todoColumnName == "" {
		log.Fatalf("Board '%s' does not have a 'Todo' column (or variation like 'To-do', 'TO-DO', etc.). Please create one first.", board.Name)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "mkanban-new-*.md")
	if err != nil {
		log.Fatalf("Failed to create temporary file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write initial content to temp file
	if _, err := tmpFile.WriteString("# "); err != nil {
		tmpFile.Close()
		log.Fatalf("Failed to write to temporary file: %v", err)
	}
	tmpFile.Close()

	// Get editor from environment or use default
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Open editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run editor: %v", err)
	}

	// Read the edited file
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		log.Fatalf("Failed to read temporary file: %v", err)
	}

	// Parse markdown to extract title and description
	title, description, err := parseMarkdownTask(string(content))
	if err != nil {
		log.Fatalf("Failed to parse task: %v", err)
	}

	// Create the task
	taskDTO, err := container.CreateTaskUseCase.Execute(ctx, boardID, dto.CreateTaskRequest{
		Title:       title,
		Description: description,
		ColumnName:  todoColumnName,
		Priority:    "",
		Tags:        []string{},
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}

	// Output full details
	fmt.Printf("Created task: %s - %s in %s (%s)\n", taskDTO.ShortID, taskDTO.Title, todoColumnName, board.Name)
}

// parseMarkdownTask extracts title and description from markdown content
func parseMarkdownTask(content string) (string, string, error) {
	lines := strings.Split(content, "\n")

	var title string
	var descriptionLines []string
	foundTitle := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Look for the first heading
		if !foundTitle && strings.HasPrefix(trimmedLine, "#") {
			// Extract title (everything after # and spaces)
			title = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "#"))
			foundTitle = true
			continue
		}

		// Everything else is description
		if foundTitle {
			descriptionLines = append(descriptionLines, line)
		}
	}

	if !foundTitle || title == "" {
		return "", "", fmt.Errorf("no title found. Please add a line starting with '# ' followed by the task title")
	}

	description := strings.TrimSpace(strings.Join(descriptionLines, "\n"))

	return title, description, nil
}

// isTodoColumn checks if a column name is a variation of "Todo"
func isTodoColumn(name string) bool {
	// Normalize: remove spaces, hyphens, and convert to lowercase
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "-", ""), " ", ""))
	return normalized == "todo"
}

// handleMigrate handles the "migrate" command
func handleMigrate(ctx context.Context, container *di.Container) {
	fmt.Println("Starting column migration to new format...")

	// Get all boards
	boards, err := container.ListBoardsUseCase.Execute(ctx)
	if err != nil {
		log.Fatalf("Failed to list boards: %v", err)
	}

	if len(boards) == 0 {
		fmt.Println("No boards found. Nothing to migrate.")
		return
	}

	migratedCount := 0
	for _, board := range boards {
		fmt.Printf("Migrating board: %s (%s)\n", board.Name, board.ID)

		// Get the board repository from the container
		boardRepo := container.BoardRepo

		// Cast to filesystem implementation to access migration method
		if fsRepo, ok := boardRepo.(interface {
			MigrateColumnsToNewFormat(context.Context, string) error
		}); ok {
			err := fsRepo.MigrateColumnsToNewFormat(ctx, board.ID)
			if err != nil {
				fmt.Printf("  Error migrating board %s: %v\n", board.ID, err)
			} else {
				fmt.Printf("  ✓ Board %s migrated successfully\n", board.ID)
				migratedCount++
			}
		} else {
			fmt.Printf("  Warning: Board repository does not support migration\n")
		}
	}

	fmt.Printf("\nMigration complete. Migrated %d board(s).\n", migratedCount)
}

// handleCheckout handles the "checkout" command
func handleCheckout(ctx context.Context, container *di.Container, cfg *config.Config, args []string) {
	// Parse arguments
	if len(args) == 0 {
		log.Fatalf("Usage: mkanban checkout <task-id> [--branch-format <format>]")
	}

	taskID := args[0]
	branchFormat := "{id}" // Default format

	// Parse optional --branch-format flag
	for i := 1; i < len(args); i++ {
		if args[i] == "--branch-format" && i+1 < len(args) {
			branchFormat = args[i+1]
			break
		}
	}

	// Determine which board to use
	var boardID string

	// Try to get active board from daemon if running in tmux
	if isRunningInTmux() {
		activeBoardID, err := getActiveBoardFromDaemon(cfg)
		if err == nil && activeBoardID != "" {
			// Verify the board exists
			_, err := container.GetBoardUseCase.Execute(ctx, activeBoardID)
			if err == nil {
				boardID = activeBoardID
			}
		}
	}

	// If no active board was found, use the first board
	if boardID == "" {
		boards, err := container.ListBoardsUseCase.Execute(ctx)
		if err != nil {
			log.Fatalf("Failed to list boards: %v", err)
		}

		if len(boards) == 0 {
			log.Fatalf("No boards found. Create a board first by running mkanban without arguments.")
		}

		boardID = boards[0].ID
	}

	// Execute the checkout use case
	err := container.CheckoutTaskUseCase.Execute(ctx, boardID, taskID, branchFormat)
	if err != nil {
		log.Fatalf("Failed to checkout task: %v", err)
	}
}
