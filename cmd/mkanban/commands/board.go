package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// boardCmd represents the board command
var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Manage kanban boards",
	Long: `Manage your kanban boards - create, list, view, delete, and switch between boards.

A board is a collection of columns and tasks. You can have multiple boards for
different projects or workflows.

Examples:
  # List all boards
  mkanban board list

  # Get details of a specific board
  mkanban board get my-project

  # Create a new board
  mkanban board create my-project --name "My Project" --description "Project tasks"

  # Create a board with default columns
  mkanban board create my-project --columns "Todo,In Progress,Review,Done"

  # Delete a board
  mkanban board delete my-project

  # Show current active board
  mkanban board current

  # Switch to a different board (for current session)
  mkanban board switch my-project`,
}

// boardListCmd lists all boards
var boardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all boards",
	Long: `List all available kanban boards.

Displays board ID, name, description, and number of tasks.

Examples:
  # List boards in text format
  mkanban board list

  # List boards in JSON format
  mkanban board list --output json

  # List boards in YAML format
  mkanban board list --output yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()

		boards, err := container.ListBoardsUseCase.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list boards: %w", err)
		}

		// Format output based on output format
		switch outputFormat {
		case "json", "yaml":
			return formatter.Print(boards)
		default:
			// Text format
			if len(boards) == 0 {
				printer.Info("No boards found. Create one with: mkanban board create <name>")
				return nil
			}

			printer.Header("Boards")
			fmt.Println()

			headers := []string{"ID", "Name", "Description", "Columns", "Tasks"}
			rows := make([][]string, 0, len(boards))

			for _, board := range boards {
				taskCount := 0
				for _, col := range board.Columns {
					taskCount += len(col.Tasks)
				}

				desc := board.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}

				rows = append(rows, []string{
					board.ID,
					board.Name,
					desc,
					fmt.Sprintf("%d", len(board.Columns)),
					fmt.Sprintf("%d", taskCount),
				})
			}

			printer.Table(headers, rows)
			return nil
		}
	},
}

// boardGetCmd gets a specific board
var boardGetCmd = &cobra.Command{
	Use:   "get <board-id>",
	Short: "Get board details",
	Long: `Get detailed information about a specific board.

Shows board metadata, columns, and tasks.

Examples:
  # Get board details
  mkanban board get my-project

  # Get board in JSON format
  mkanban board get my-project --output json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		boardID := args[0]

		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		// Format output
		switch outputFormat {
		case "json", "yaml":
			return formatter.Print(board)
		default:
			// Text format
			printer.Header(board.Name)
			fmt.Println()
			printer.Println("ID:          %s", board.ID)
			printer.Println("Description: %s", board.Description)
			printer.Println("Columns:     %d", len(board.Columns))

			taskCount := 0
			for _, col := range board.Columns {
				taskCount += len(col.Tasks)
			}
			printer.Println("Tasks:       %d", taskCount)
			fmt.Println()

			// List columns
			if len(board.Columns) > 0 {
				printer.Bold("Columns:")
				for _, col := range board.Columns {
					printer.Println("  - %s (%d tasks)", col.Name, len(col.Tasks))
				}
			}

			return nil
		}
	},
}

// boardCreateCmd creates a new board
var boardCreateCmd = &cobra.Command{
	Use:   "create <board-id>",
	Short: "Create a new board",
	Long: `Create a new kanban board.

You can specify the board name, description, task prefix, and initial columns.

Examples:
  # Create a basic board
  mkanban board create my-project --name "My Project"

  # Create a board with description
  mkanban board create my-project \
    --name "My Project" \
    --description "Project management tasks"

  # Create a board with custom task prefix
  mkanban board create my-project \
    --name "My Project" \
    --prefix "PROJ"

  # Create a board with initial columns
  mkanban board create my-project \
    --name "My Project" \
    --columns "Backlog,Todo,In Progress,Review,Done"

  # Create a board for a specific repository
  mkanban board create my-project \
    --name "My Project" \
    --repo-path "/path/to/repo"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		boardID := args[0]

		// Get flags
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		prefix, _ := cmd.Flags().GetString("prefix")
		repoPath, _ := cmd.Flags().GetString("repo-path")
		columnsStr, _ := cmd.Flags().GetString("columns")

		// Default name to board ID if not provided
		if name == "" {
			name = boardID
		}

		// Create board
		board, err := container.CreateBoardUseCase.Execute(ctx, struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}{
			Name:        name,
			Description: description,
		})
		if err != nil {
			return fmt.Errorf("failed to create board: %w", err)
		}

		// Create default columns if specified
		if columnsStr != "" {
			columns := strings.Split(columnsStr, ",")
			for i, colName := range columns {
				colName = strings.TrimSpace(colName)
				_, err := container.CreateColumnUseCase.Execute(ctx, board.ID, struct {
					Name        string  `json:"name"`
					Description string  `json:"description"`
					Order       int     `json:"order"`
					WIPLimit    int     `json:"wip_limit"`
					Color       *string `json:"color,omitempty"`
				}{
					Name:     colName,
					Order:    i + 1,
					WIPLimit: 0, // Unlimited
				})
				if err != nil {
					return fmt.Errorf("failed to create column %s: %w", colName, err)
				}
			}
		}

		printer.Success("Created board: %s (%s)", board.Name, board.ID)
		if columnsStr != "" {
			printer.Info("Created %d columns", len(strings.Split(columnsStr, ",")))
		}

		// Note about unused flags (for future implementation)
		if prefix != "" || repoPath != "" {
			printer.Warning("Note: --prefix and --repo-path flags are not yet implemented")
		}

		return nil
	},
}

// boardDeleteCmd deletes a board
var boardDeleteCmd = &cobra.Command{
	Use:   "delete <board-id>",
	Short: "Delete a board",
	Long: `Delete a kanban board and all its columns and tasks.

WARNING: This action cannot be undone. All tasks and data in the board will be permanently deleted.

Examples:
  # Delete a board (with confirmation prompt)
  mkanban board delete my-project

  # Delete a board without confirmation
  mkanban board delete my-project --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		boardID := args[0]

		force, _ := cmd.Flags().GetBool("force")

		// Get board to show what will be deleted
		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		// Count tasks
		taskCount := 0
		for _, col := range board.Columns {
			taskCount += len(col.Tasks)
		}

		// Confirm deletion unless --force is used
		if !force {
			printer.Warning("About to delete board '%s' with %d columns and %d tasks", board.Name, len(board.Columns), taskCount)
			printer.Warning("This action cannot be undone!")
			fmt.Print("\nType the board ID to confirm: ")

			var confirmation string
			fmt.Scanln(&confirmation)

			if confirmation != boardID {
				printer.Info("Deletion cancelled")
				return nil
			}
		}

		// TODO: Implement board deletion use case
		// For now, just show what would be deleted
		printer.Error("Board deletion not yet implemented")
		printer.Info("Would delete: %s (%d columns, %d tasks)", board.Name, len(board.Columns), taskCount)

		return fmt.Errorf("board deletion not yet implemented")
	},
}

// boardCurrentCmd shows the current active board
var boardCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current active board",
	Long: `Show the currently active board for the current session.

The active board is determined by:
  1. Current tmux session (if running in tmux)
  2. Most recently used board
  3. First available board

Examples:
  # Show current board
  mkanban board current

  # Show current board in JSON format
  mkanban board current --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		switch outputFormat {
		case "json", "yaml":
			return formatter.Print(board)
		default:
			printer.Info("Current board: %s (%s)", board.Name, board.ID)
			return nil
		}
	},
}

// boardSwitchCmd switches the active board
var boardSwitchCmd = &cobra.Command{
	Use:   "switch <board-id>",
	Short: "Switch to a different board",
	Long: `Switch the active board for the current session.

This sets the default board for subsequent commands in the current tmux session.

Examples:
  # Switch to a different board
  mkanban board switch my-other-project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		boardID := args[0]

		// Verify board exists
		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		// TODO: Implement session tracking to set active board
		printer.Error("Board switching not yet implemented")
		printer.Info("Would switch to: %s (%s)", board.Name, board.ID)
		printer.Info("For now, use --board-id flag: mkanban --board-id %s <command>", boardID)

		return fmt.Errorf("board switching not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(boardCmd)

	// Add subcommands
	boardCmd.AddCommand(boardListCmd)
	boardCmd.AddCommand(boardGetCmd)
	boardCmd.AddCommand(boardCreateCmd)
	boardCmd.AddCommand(boardDeleteCmd)
	boardCmd.AddCommand(boardCurrentCmd)
	boardCmd.AddCommand(boardSwitchCmd)

	// boardCreateCmd flags
	boardCreateCmd.Flags().String("name", "", "Board name (default: board-id)")
	boardCreateCmd.Flags().String("description", "", "Board description")
	boardCreateCmd.Flags().String("prefix", "", "Task ID prefix (e.g., PROJ)")
	boardCreateCmd.Flags().String("repo-path", "", "Repository path for git integration")
	boardCreateCmd.Flags().String("columns", "", "Comma-separated list of initial columns")

	// boardDeleteCmd flags
	boardDeleteCmd.Flags().Bool("force", false, "Delete without confirmation")
}
