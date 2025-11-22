package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// columnCmd represents the column command
var columnCmd = &cobra.Command{
	Use:   "column",
	Short: "Manage board columns",
	Long: `Manage columns in your kanban boards - create, list, update, delete, and reorder columns.

Columns organize tasks into different stages of your workflow (e.g., Todo, In Progress, Done).

Examples:
  # List all columns in current board
  mkanban column list

  # List columns in specific board
  mkanban column list --board-id my-project

  # Get details of a specific column
  mkanban column get "In Progress"

  # Create a new column
  mkanban column create "Code Review" --position 3

  # Update column properties
  mkanban column update "In Progress" --wip-limit 5

  # Delete a column
  mkanban column delete "Archived"

  # Reorder columns
  mkanban column reorder "Backlog,Todo,In Progress,Review,Done"`,
}

// columnListCmd lists all columns in a board
var columnListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all columns",
	Long: `List all columns in a board.

Displays column name, order, WIP limit, and number of tasks.

Examples:
  # List columns in current board
  mkanban column list

  # List columns in JSON format
  mkanban column list --output json`,
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
			return formatter.Print(board.Columns)
		default:
			if len(board.Columns) == 0 {
				printer.Info("No columns found in board '%s'", board.Name)
				printer.Info("Create one with: mkanban column create <name>")
				return nil
			}

			printer.Header("Columns in %s", board.Name)
			fmt.Println()

			headers := []string{"Name", "Order", "WIP Limit", "Tasks"}
			rows := make([][]string, 0, len(board.Columns))

			for _, col := range board.Columns {
				wipLimit := "Unlimited"
				if col.WIPLimit > 0 {
					wipLimit = strconv.Itoa(col.WIPLimit)
				}

				rows = append(rows, []string{
					col.Name,
					strconv.Itoa(col.Order),
					wipLimit,
					strconv.Itoa(len(col.Tasks)),
				})
			}

			printer.Table(headers, rows)
			return nil
		}
	},
}

// columnGetCmd gets details of a specific column
var columnGetCmd = &cobra.Command{
	Use:   "get <column-name>",
	Short: "Get column details",
	Long: `Get detailed information about a specific column.

Examples:
  # Get column details
  mkanban column get "In Progress"

  # Get column in JSON format
  mkanban column get "Todo" --output json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		columnName := args[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		// Find the column
		var foundColumn *struct {
			Name        string
			Description string
			Order       int
			WIPLimit    int
			Tasks       []interface{}
		}

		for i, col := range board.Columns {
			if col.Name == columnName {
				foundColumn = &board.Columns[i]
				break
			}
		}

		if foundColumn == nil {
			return fmt.Errorf("column '%s' not found in board '%s'", columnName, board.Name)
		}

		switch outputFormat {
		case "json", "yaml":
			return formatter.Print(foundColumn)
		default:
			printer.Header(foundColumn.Name)
			fmt.Println()
			printer.Println("Order:       %d", foundColumn.Order)
			printer.Println("WIP Limit:   %s", func() string {
				if foundColumn.WIPLimit > 0 {
					return strconv.Itoa(foundColumn.WIPLimit)
				}
				return "Unlimited"
			}())
			printer.Println("Tasks:       %d", len(foundColumn.Tasks))
			printer.Println("Description: %s", foundColumn.Description)
			return nil
		}
	},
}

// columnCreateCmd creates a new column
var columnCreateCmd = &cobra.Command{
	Use:   "create <column-name>",
	Short: "Create a new column",
	Long: `Create a new column in a board.

You can specify the column position, WIP limit, and description.

Examples:
  # Create a basic column
  mkanban column create "Code Review"

  # Create a column at specific position
  mkanban column create "Testing" --position 4

  # Create a column with WIP limit
  mkanban column create "In Progress" --wip-limit 5

  # Create a column with description
  mkanban column create "Done" --description "Completed tasks"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		columnName := args[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get flags
		description, _ := cmd.Flags().GetString("description")
		position, _ := cmd.Flags().GetInt("position")
		wipLimit, _ := cmd.Flags().GetInt("wip-limit")

		// If position is not specified, add to end
		if position == 0 {
			board, err := container.GetBoardUseCase.Execute(ctx, boardID)
			if err != nil {
				return fmt.Errorf("failed to get board: %w", err)
			}
			position = len(board.Columns) + 1
		}

		// Create column
		_, err = container.CreateColumnUseCase.Execute(ctx, boardID, struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Order       int     `json:"order"`
			WIPLimit    int     `json:"wip_limit"`
			Color       *string `json:"color,omitempty"`
		}{
			Name:        columnName,
			Description: description,
			Order:       position,
			WIPLimit:    wipLimit,
		})
		if err != nil {
			return fmt.Errorf("failed to create column: %w", err)
		}

		printer.Success("Created column: %s", columnName)
		if wipLimit > 0 {
			printer.Info("WIP limit: %d", wipLimit)
		}

		return nil
	},
}

// columnUpdateCmd updates a column
var columnUpdateCmd = &cobra.Command{
	Use:   "update <column-name>",
	Short: "Update column properties",
	Long: `Update properties of an existing column.

You can update the column name, WIP limit, position, or description.

Examples:
  # Update column name
  mkanban column update "In Progress" --name "Working On"

  # Update WIP limit
  mkanban column update "In Progress" --wip-limit 5

  # Remove WIP limit
  mkanban column update "In Progress" --wip-limit 0

  # Update position
  mkanban column update "Review" --position 3

  # Update description
  mkanban column update "Done" --description "Completed and deployed"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		columnName := args[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get flags
		newName, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		position, _ := cmd.Flags().GetInt("position")
		wipLimit, _ := cmd.Flags().GetInt("wip-limit")

		// Check if any flags were provided
		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") &&
		   !cmd.Flags().Changed("position") && !cmd.Flags().Changed("wip-limit") {
			return fmt.Errorf("no updates specified. Use --name, --description, --position, or --wip-limit")
		}

		// TODO: Implement column update use case
		printer.Error("Column update not yet implemented")
		printer.Info("Would update column '%s' in board '%s':", columnName, boardID)
		if newName != "" {
			printer.Info("  New name: %s", newName)
		}
		if cmd.Flags().Changed("description") {
			printer.Info("  Description: %s", description)
		}
		if position > 0 {
			printer.Info("  Position: %d", position)
		}
		if cmd.Flags().Changed("wip-limit") {
			printer.Info("  WIP limit: %d", wipLimit)
		}

		return fmt.Errorf("column update not yet implemented")
	},
}

// columnDeleteCmd deletes a column
var columnDeleteCmd = &cobra.Command{
	Use:   "delete <column-name>",
	Short: "Delete a column",
	Long: `Delete a column from a board.

WARNING: By default, this will fail if the column contains tasks.
Use --move-tasks-to to move tasks to another column before deletion.

Examples:
  # Delete an empty column
  mkanban column delete "Archived"

  # Delete column and move tasks to another column
  mkanban column delete "Archived" --move-tasks-to "Done"

  # Force delete column with tasks (tasks will be deleted)
  mkanban column delete "Archived" --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		columnName := args[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		force, _ := cmd.Flags().GetBool("force")
		moveTasksTo, _ := cmd.Flags().GetString("move-tasks-to")

		// Get board to check column
		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		// Find the column
		var foundColumn *struct {
			Name  string
			Tasks []interface{}
		}

		for i, col := range board.Columns {
			if col.Name == columnName {
				foundColumn = &board.Columns[i]
				break
			}
		}

		if foundColumn == nil {
			return fmt.Errorf("column '%s' not found in board '%s'", columnName, board.Name)
		}

		// Check if column has tasks
		if len(foundColumn.Tasks) > 0 && !force && moveTasksTo == "" {
			return fmt.Errorf("column '%s' contains %d tasks. Use --move-tasks-to or --force", columnName, len(foundColumn.Tasks))
		}

		// TODO: Implement column deletion
		printer.Error("Column deletion not yet implemented")
		printer.Info("Would delete column '%s' from board '%s' (%d tasks)", columnName, board.Name, len(foundColumn.Tasks))
		if moveTasksTo != "" {
			printer.Info("Would move tasks to column '%s'", moveTasksTo)
		}

		return fmt.Errorf("column deletion not yet implemented")
	},
}

// columnReorderCmd reorders columns
var columnReorderCmd = &cobra.Command{
	Use:   "reorder <col1,col2,col3,...>",
	Short: "Reorder columns",
	Long: `Reorder columns in a board.

Provide a comma-separated list of column names in the desired order.

Examples:
  # Reorder columns
  mkanban column reorder "Backlog,Todo,In Progress,Review,Done"

  # Reorder with spaces in column names
  mkanban column reorder "To Do,In Progress,Code Review,Done"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		columnsStr := args[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Parse column names
		columnNames := strings.Split(columnsStr, ",")
		for i, name := range columnNames {
			columnNames[i] = strings.TrimSpace(name)
		}

		// TODO: Implement column reordering
		printer.Error("Column reordering not yet implemented")
		printer.Info("Would reorder columns in board '%s':", boardID)
		for i, name := range columnNames {
			printer.Info("  %d. %s", i+1, name)
		}

		return fmt.Errorf("column reordering not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(columnCmd)

	// Add subcommands
	columnCmd.AddCommand(columnListCmd)
	columnCmd.AddCommand(columnGetCmd)
	columnCmd.AddCommand(columnCreateCmd)
	columnCmd.AddCommand(columnUpdateCmd)
	columnCmd.AddCommand(columnDeleteCmd)
	columnCmd.AddCommand(columnReorderCmd)

	// columnCreateCmd flags
	columnCreateCmd.Flags().String("description", "", "Column description")
	columnCreateCmd.Flags().Int("position", 0, "Column position (default: end)")
	columnCreateCmd.Flags().Int("wip-limit", 0, "WIP limit (0 = unlimited)")

	// columnUpdateCmd flags
	columnUpdateCmd.Flags().String("name", "", "New column name")
	columnUpdateCmd.Flags().String("description", "", "Column description")
	columnUpdateCmd.Flags().Int("position", 0, "Column position")
	columnUpdateCmd.Flags().Int("wip-limit", 0, "WIP limit (0 = unlimited)")

	// columnDeleteCmd flags
	columnDeleteCmd.Flags().Bool("force", false, "Force delete even if column has tasks")
	columnDeleteCmd.Flags().String("move-tasks-to", "", "Move tasks to this column before deletion")
}
