package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"mkanban/internal/application/dto"
	"mkanban/internal/infrastructure/serialization"
)

// taskCmd represents the task command
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long: `Manage tasks in your kanban boards - create, update, move, delete, and query tasks.

Tasks are the core work items in your kanban board. Each task has a unique ID,
title, description, priority, status, tags, and optional due date.

Examples:
  # List all tasks
  mkanban task list

  # List tasks in a specific column
  mkanban task list --column "In Progress"

  # Create a new task
  mkanban task create --title "Fix login bug" --priority high

  # Update a task
  mkanban task update TASK-123 --priority critical --add-tag urgent

  # Move task to next column
  mkanban task advance TASK-123

  # Delete a task
  mkanban task delete TASK-123

  # Checkout git branch for task
  mkanban task checkout TASK-123`,
}

// taskListCmd lists tasks
var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long: `List tasks with optional filtering.

You can filter by column, priority, status, tags, due date, and more.

Output formats:
  text - Human-readable table (default)
  json - JSON output for scripting
  yaml - YAML output
  fzf  - Task ID and title (tab-separated)
  path - File paths with titles (format: path :: title)

Examples:
  # List all tasks
  mkanban task list

  # List tasks in specific column
  mkanban task list --column "Todo"

  # List high priority tasks
  mkanban task list --priority high

  # List overdue tasks
  mkanban task list --overdue

  # List tasks due before a date
  mkanban task list --due-before "2025-12-31"

  # List tasks with specific tag
  mkanban task list --tag urgent

  # List tasks across all boards
  mkanban task list --all-boards

  # List in path format (for scripting)
  mkanban task list --output path

  # Pipe to fzf and checkout the selected task
  mkanban task list --output fzf | fzf | mkanban task checkout

  # Combine filters
  mkanban task list --column "In Progress" --priority high --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()

		// Get filters
		column, _ := cmd.Flags().GetString("column")
		priority, _ := cmd.Flags().GetString("priority")
		status, _ := cmd.Flags().GetString("status")
		tag, _ := cmd.Flags().GetString("tag")
		overdue, _ := cmd.Flags().GetBool("overdue")
		dueBefore, _ := cmd.Flags().GetString("due-before")
		allBoards, _ := cmd.Flags().GetBool("all-boards")

		matchesFilters := func(task dto.TaskDTO) (bool, error) {
			if column != "" && task.ColumnName != column {
				return false, nil
			}
			if priority != "" && task.Priority != priority {
				return false, nil
			}
			if status != "" && task.Status != status {
				return false, nil
			}
			if tag != "" {
				hasTag := false
				for _, t := range task.Tags {
					if t == tag {
						hasTag = true
						break
					}
				}
				if !hasTag {
					return false, nil
				}
			}
			if overdue {
				if task.DueDate == nil {
					return false, nil
				}
				if time.Now().Before(*task.DueDate) {
					return false, nil
				}
			}
			if dueBefore != "" {
				if task.DueDate == nil {
					return false, nil
				}
				dueBeforeDate, err := time.Parse("2006-01-02", dueBefore)
				if err != nil {
					return false, fmt.Errorf("invalid due-before date format. Use YYYY-MM-DD: %w", err)
				}
				if task.DueDate.After(dueBeforeDate) {
					return false, nil
				}
			}
			return true, nil
		}

		type taskWithBoard struct {
			dto.TaskDTO
			BoardID   string `json:"board_id"`
			BoardName string `json:"board_name"`
		}

		filteredTasks := make([]dto.TaskDTO, 0)
		filteredTasksWithBoard := make([]taskWithBoard, 0)
		var boards []dto.BoardListDTO

		if allBoards {
			var err error
			boards, err = container.ListBoardsUseCase.Execute(ctx)
			if err != nil {
				return fmt.Errorf("failed to list boards: %w", err)
			}

			for _, board := range boards {
				tasks, err := container.ListTasksUseCase.Execute(ctx, board.ID)
				if err != nil {
					return fmt.Errorf("failed to list tasks for board %s: %w", board.ID, err)
				}

				for _, task := range tasks {
					matches, err := matchesFilters(task)
					if err != nil {
						return err
					}
					if !matches {
						continue
					}
					filteredTasksWithBoard = append(filteredTasksWithBoard, taskWithBoard{
						TaskDTO:   task,
						BoardID:   board.ID,
						BoardName: board.Name,
					})
				}
			}
		} else {
			boardID, err := getBoardID(ctx)
			if err != nil {
				return err
			}

			tasks, err := container.ListTasksUseCase.Execute(ctx, boardID)
			if err != nil {
				return fmt.Errorf("failed to list tasks: %w", err)
			}

			for _, task := range tasks {
				matches, err := matchesFilters(task)
				if err != nil {
					return err
				}
				if !matches {
					continue
				}
				filteredTasks = append(filteredTasks, task)
			}
		}

		// Format output
		switch outputFormat {
		case "fzf":
			if allBoards {
				for _, task := range filteredTasksWithBoard {
					fmt.Printf("%s\t%s\t[%s]\n", task.ShortID, task.Title, task.BoardName)
				}
				return nil
			}
			for _, task := range filteredTasks {
				fmt.Printf("%s\t%s\n", task.ShortID, task.Title)
			}
			return nil
		case "path":
			if allBoards {
				for _, task := range filteredTasksWithBoard {
					fmt.Printf("%s :: %s :: %s\n", task.FilePath, task.Title, task.BoardName)
				}
				return nil
			}
			for _, task := range filteredTasks {
				fmt.Printf("%s :: %s\n", task.FilePath, task.Title)
			}
			return nil
		case "json", "yaml":
			if allBoards {
				return formatter.Print(filteredTasksWithBoard)
			}
			return formatter.Print(filteredTasks)
		default:
			// Text format
			if allBoards {
				if len(filteredTasksWithBoard) == 0 {
					printer.Info("No tasks found")
					return nil
				}

				tasksByBoard := make(map[string][]dto.TaskDTO)
				for _, task := range filteredTasksWithBoard {
					tasksByBoard[task.BoardID] = append(tasksByBoard[task.BoardID], task.TaskDTO)
				}

				for _, board := range boards {
					boardTasks := tasksByBoard[board.ID]
					if len(boardTasks) == 0 {
						continue
					}

					printer.Header("%s (%s)", board.Name, board.ID)
					fmt.Println()
					printTaskColumns(boardTasks)
					printer.Info("Total: %d tasks", len(boardTasks))
					fmt.Println()
				}
				return nil
			}

			if len(filteredTasks) == 0 {
				printer.Info("No tasks found")
				return nil
			}

			printTaskColumns(filteredTasks)

			printer.Info("Total: %d tasks", len(filteredTasks))
			return nil
		}
	},
}

func printTaskColumns(tasks []dto.TaskDTO) {
	columns := make(map[string][]dto.TaskDTO)
	for _, task := range tasks {
		columns[task.ColumnName] = append(columns[task.ColumnName], task)
	}

	for colName, colTasks := range columns {
		printer.Header(colName)
		fmt.Println()

		for _, task := range colTasks {
			priorityIcon := "⚪"
			if task.Priority == "high" || task.Priority == "critical" {
				priorityIcon = "⚫"
			}

			dueInfo := ""
			if task.DueDate != nil {
				daysUntil := int(time.Until(*task.DueDate).Hours() / 24)
				if daysUntil < 0 {
					dueInfo = fmt.Sprintf("📅 overdue %d days", -daysUntil)
				} else if daysUntil == 0 {
					dueInfo = "📅 due today"
				} else if daysUntil == 1 {
					dueInfo = "📅 due tomorrow"
				} else if daysUntil < 7 {
					dueInfo = fmt.Sprintf("📅 due in %d days", daysUntil)
				}
			}

			tagsInfo := ""
			if len(task.Tags) > 0 {
				tagsInfo = "🏷️  " + strings.Join(task.Tags, ", ")
			}

			printer.Println("  %s %s %s", priorityIcon, task.ShortID, task.Title)
			if dueInfo != "" {
				printer.Subtle("      %s", dueInfo)
			}
			if tagsInfo != "" {
				printer.Subtle("      %s", tagsInfo)
			}
			fmt.Println()
		}
	}
}

// taskGetCmd gets a specific task
var taskGetCmd = &cobra.Command{
	Use:   "get <task-id>",
	Short: "Get task details",
	Long: `Get detailed information about a specific task.

Examples:
  # Get task details
  mkanban task get TASK-123

  # Get task in JSON format
  mkanban task get TASK-123 --output json

  # Get task in markdown format
  mkanban task get TASK-123 --output markdown`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get all tasks and find the one we want
		tasks, err := container.ListTasksUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		var foundTask *dto.TaskDTO
		for i, task := range tasks {
			if task.ShortID == taskID || task.ID == taskID {
				foundTask = &tasks[i]
				break
			}
		}

		if foundTask == nil {
			return fmt.Errorf("task '%s' not found", taskID)
		}

		// Format output
		switch outputFormat {
		case "json", "yaml":
			return formatter.Print(foundTask)
		case "markdown":
			// Markdown format with frontmatter
			fmt.Printf("---\n")
			fmt.Printf("id: %s\n", foundTask.ID)
			fmt.Printf("title: %s\n", foundTask.Title)
			fmt.Printf("priority: %s\n", foundTask.Priority)
			fmt.Printf("status: %s\n", foundTask.Status)
			if len(foundTask.Tags) > 0 {
				fmt.Printf("tags: [%s]\n", strings.Join(foundTask.Tags, ", "))
			}
			if foundTask.DueDate != nil {
				fmt.Printf("due: %s\n", foundTask.DueDate.Format("2006-01-02"))
			}
			fmt.Printf("---\n\n")
			fmt.Printf("# %s\n\n", foundTask.Title)
			fmt.Printf("%s\n", foundTask.Description)
			return nil
		default:
			// Text format
			printer.Header(foundTask.Title)
			fmt.Println()
			printer.Println("ID:          %s", foundTask.ShortID)
			printer.Println("Full ID:     %s", foundTask.ID)
			printer.Println("Column:      %s", foundTask.ColumnName)
			printer.Println("Priority:    %s", foundTask.Priority)
			printer.Println("Status:      %s", foundTask.Status)
			if len(foundTask.Tags) > 0 {
				printer.Println("Tags:        %s", strings.Join(foundTask.Tags, ", "))
			}
			if foundTask.DueDate != nil {
				printer.Println("Due:         %s", foundTask.DueDate.Format("2006-01-02"))
			}
			printer.Println("Path:        %s", foundTask.FilePath)
			fmt.Println()
			if foundTask.Description != "" {
				printer.Bold("Description:")
				fmt.Println(foundTask.Description)
			}
			return nil
		}
	},
}

// taskCreateCmd creates a new task
var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	Long: `Create a new task in a board.

You can specify the title, description, priority, column, tags, and due date.
Use the --edit flag to open an editor for the description.

Examples:
  # Create a basic task
  mkanban task create --title "Fix login bug"

  # Create a task with details
  mkanban task create \
    --title "Implement dark mode" \
    --description "Add dark mode theme to UI" \
    --priority high \
    --column "Todo" \
    --tags "frontend,ui"

  # Create a task with due date
  mkanban task create --title "Review PR" --due "2025-12-25"

  # Create a task with editor for description
  mkanban task create --title "Write documentation" --edit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get flags
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		column, _ := cmd.Flags().GetString("column")
		priority, _ := cmd.Flags().GetString("priority")
		status, _ := cmd.Flags().GetString("status")
		tagsStr, _ := cmd.Flags().GetString("tags")
		dueStr, _ := cmd.Flags().GetString("due")
		useEditor, _ := cmd.Flags().GetBool("edit")

		var tags []string

		if title == "" {
			parsedTags := parseTagsString(tagsStr)
			editedContent, err := openEditorForEmptyTask(priority, parsedTags)
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}

			doc, err := serialization.ParseFrontmatter([]byte(editedContent))
			if err != nil {
				return fmt.Errorf("failed to parse task frontmatter: %w", err)
			}

			parsedTitle, parsedDesc, err := parseMarkdownTask(doc.Content)
			if err != nil {
				return fmt.Errorf("failed to parse task: %w", err)
			}
			title = parsedTitle
			description = parsedDesc

			if priority == "" {
				priority = doc.GetString("priority")
			}

			if tagsStr == "" {
				tags = doc.GetStringSlice("tags")
			} else {
				tags = parsedTags
			}
		} else if useEditor {
			editedContent, err := openEditorForTask(title, description)
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}
			// Parse markdown content
			parsedTitle, parsedDesc, err := parseMarkdownTask(editedContent)
			if err != nil {
				return fmt.Errorf("failed to parse task: %w", err)
			}
			title = parsedTitle
			description = parsedDesc
		}

		// Default column to "Todo" if not specified
		if column == "" {
			// Find a "Todo" column variation
			board, err := container.GetBoardUseCase.Execute(ctx, boardID)
			if err != nil {
				return fmt.Errorf("failed to get board: %w", err)
			}

			for _, col := range board.Columns {
				if isTodoColumn(col.Name) {
					column = col.Name
					break
				}
			}

			if column == "" && len(board.Columns) > 0 {
				// Use first column
				column = board.Columns[0].Name
			}

			if column == "" {
				return fmt.Errorf("no columns found in board. Create a column first")
			}
		}

		// Parse tags
		if title != "" && tags == nil {
			tags = parseTagsString(tagsStr)
		}

		// Parse due date
		// var dueDate *time.Time
		// if dueStr != "" {
		// 	parsedDate, err := time.Parse("2006-01-02", dueStr)
		// 	if err != nil {
		// 		return fmt.Errorf("invalid due date format. Use YYYY-MM-DD: %w", err)
		// 	}
		// 	dueDate = &parsedDate
		// }

		// Create task
		task, err := container.CreateTaskUseCase.Execute(ctx, boardID, dto.CreateTaskRequest{
			Title:       title,
			Description: description,
			ColumnName:  column,
			Priority:    priority,
			Tags:        tags,
			// DueDate:     dueDate, // TODO: Add when DTO supports it
		})

		// Note: Status parameter ignored as CreateTaskRequest doesn't support it
		_ = status
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		printer.Success("Created task: %s - %s", task.ShortID, task.Title)
		printer.Info("Column: %s", column)
		if priority != "" {
			printer.Info("Priority: %s", priority)
		}
		if len(tags) > 0 {
			printer.Info("Tags: %s", strings.Join(tags, ", "))
		}
		if dueStr != "" {
			printer.Info("Due: %s", dueStr)
		}

		return nil
	},
}

// taskUpdateCmd updates a task
var taskUpdateCmd = &cobra.Command{
	Use:   "update <task-id>",
	Short: "Update task properties",
	Long: `Update properties of an existing task.

You can update title, description, priority, status, tags, and due date.

Examples:
  # Update task title
  mkanban task update TASK-123 --title "New title"

  # Update priority
  mkanban task update TASK-123 --priority critical

  # Add tags
  mkanban task update TASK-123 --add-tag urgent --add-tag bug

  # Remove tags
  mkanban task update TASK-123 --remove-tag wontfix

  # Set tags (replaces all existing tags)
  mkanban task update TASK-123 --tags "frontend,ui,urgent"

  # Update due date
  mkanban task update TASK-123 --due "2025-12-31"

  # Edit description in editor
  mkanban task update TASK-123 --edit`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get flags
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		priority, _ := cmd.Flags().GetString("priority")
		status, _ := cmd.Flags().GetString("status")
		tagsStr, _ := cmd.Flags().GetString("tags")
		addTags, _ := cmd.Flags().GetStringSlice("add-tag")
		removeTags, _ := cmd.Flags().GetStringSlice("remove-tag")
		dueStr, _ := cmd.Flags().GetString("due")
		useEditor, _ := cmd.Flags().GetBool("edit")

		// Check if any flags were provided
		if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("description") &&
			!cmd.Flags().Changed("priority") && !cmd.Flags().Changed("status") &&
			!cmd.Flags().Changed("tags") && len(addTags) == 0 && len(removeTags) == 0 &&
			!cmd.Flags().Changed("due") && !useEditor {
			return fmt.Errorf("no updates specified")
		}

		// TODO: Implement task update use case
		printer.Error("Task update not yet implemented")
		printer.Info("Would update task '%s' in board '%s':", taskID, boardID)
		if title != "" {
			printer.Info("  Title: %s", title)
		}
		if cmd.Flags().Changed("description") {
			printer.Info("  Description: %s", description)
		}
		if priority != "" {
			printer.Info("  Priority: %s", priority)
		}
		if status != "" {
			printer.Info("  Status: %s", status)
		}
		if tagsStr != "" {
			printer.Info("  Tags: %s", tagsStr)
		}
		if len(addTags) > 0 {
			printer.Info("  Add tags: %s", strings.Join(addTags, ", "))
		}
		if len(removeTags) > 0 {
			printer.Info("  Remove tags: %s", strings.Join(removeTags, ", "))
		}
		if dueStr != "" {
			printer.Info("  Due: %s", dueStr)
		}
		if useEditor {
			printer.Info("  Open editor: yes")
		}

		return fmt.Errorf("task update not yet implemented")
	},
}

// taskMoveCmd moves a task to a different column
var taskMoveCmd = &cobra.Command{
	Use:   "move <task-id> <column-name>",
	Short: "Move task to a specific column",
	Long: `Move a task to a specific column.

This is the CLI equivalent of the TUI 'm' or Enter key action.

Examples:
  # Move task to "In Progress"
  mkanban task move TASK-123 "In Progress"

  # Move task to "Done"
  mkanban task move TASK-123 Done`,
	Args: cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 2)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]
		targetColumn := resolvedArgs[1]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Execute move task use case
		moveReq := dto.MoveTaskRequest{
			TaskID:           taskID,
			TargetColumnName: targetColumn,
		}

		_, err = container.MoveTaskUseCase.Execute(ctx, boardID, moveReq)
		if err != nil {
			return fmt.Errorf("failed to move task: %w", err)
		}

		printer.Success("Moved task %s to %s", taskID, targetColumn)
		return nil
	},
}

// taskAdvanceCmd moves a task to the next column
var taskAdvanceCmd = &cobra.Command{
	Use:   "advance <task-id>",
	Short: "Move task to next column",
	Long: `Move a task to the next column in the board.

This is the CLI equivalent of the TUI 'm' or Enter key action.

Examples:
  # Move task to next column
  mkanban task advance TASK-123`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get board to determine columns
		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		// Find task and its current column
		var currentColumn string
		var task *dto.TaskDTO
		for _, col := range board.Columns {
			for i, t := range col.Tasks {
				if t.ShortID == taskID || t.ID == taskID {
					currentColumn = col.Name
					task = &col.Tasks[i]
					break
				}
			}
			if task != nil {
				break
			}
		}

		if task == nil {
			return fmt.Errorf("task '%s' not found", taskID)
		}

		// Find next column
		var nextColumn string
		for i, col := range board.Columns {
			if col.Name == currentColumn && i+1 < len(board.Columns) {
				nextColumn = board.Columns[i+1].Name
				break
			}
		}

		if nextColumn == "" {
			return fmt.Errorf("task is already in the last column")
		}

		// Move task
		moveReq := dto.MoveTaskRequest{
			TaskID:           taskID,
			TargetColumnName: nextColumn,
		}

		_, err = container.MoveTaskUseCase.Execute(ctx, boardID, moveReq)
		if err != nil {
			return fmt.Errorf("failed to move task: %w", err)
		}

		printer.Success("Moved task %s from %s to %s", taskID, currentColumn, nextColumn)
		return nil
	},
}

// taskRetreatCmd moves a task to the previous column
var taskRetreatCmd = &cobra.Command{
	Use:   "retreat <task-id>",
	Short: "Move task to previous column",
	Long: `Move a task to the previous column in the board.

Examples:
  # Move task to previous column
  mkanban task retreat TASK-123`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get board to determine columns
		board, err := container.GetBoardUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to get board: %w", err)
		}

		// Find task and its current column
		var currentColumn string
		var task *dto.TaskDTO
		for _, col := range board.Columns {
			for i, t := range col.Tasks {
				if t.ShortID == taskID || t.ID == taskID {
					currentColumn = col.Name
					task = &col.Tasks[i]
					break
				}
			}
			if task != nil {
				break
			}
		}

		if task == nil {
			return fmt.Errorf("task '%s' not found", taskID)
		}

		// Find previous column
		var prevColumn string
		for i, col := range board.Columns {
			if col.Name == currentColumn && i > 0 {
				prevColumn = board.Columns[i-1].Name
				break
			}
		}

		if prevColumn == "" {
			return fmt.Errorf("task is already in the first column")
		}

		// Move task
		moveReq := dto.MoveTaskRequest{
			TaskID:           taskID,
			TargetColumnName: prevColumn,
		}

		_, err = container.MoveTaskUseCase.Execute(ctx, boardID, moveReq)
		if err != nil {
			return fmt.Errorf("failed to move task: %w", err)
		}

		printer.Success("Moved task %s from %s to %s", taskID, currentColumn, prevColumn)
		return nil
	},
}

// taskDeleteCmd deletes a task
var taskDeleteCmd = &cobra.Command{
	Use:   "delete <task-id>",
	Short: "Delete a task",
	Long: `Delete a task from the board.

This is the CLI equivalent of the TUI 'd' key action.

WARNING: This action cannot be undone.

Examples:
  # Delete a task (with confirmation)
  mkanban task delete TASK-123

  # Delete without confirmation
  mkanban task delete TASK-123 --force`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		force, _ := cmd.Flags().GetBool("force")

		// Get task details for confirmation
		tasks, err := container.ListTasksUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		var foundTask *dto.TaskDTO
		for i, task := range tasks {
			if task.ShortID == taskID || task.ID == taskID {
				foundTask = &tasks[i]
				break
			}
		}

		if foundTask == nil {
			return fmt.Errorf("task '%s' not found", taskID)
		}

		// Confirm deletion unless --force is used
		if !force {
			printer.Warning("About to delete task: %s - %s", foundTask.ShortID, foundTask.Title)
			printer.Warning("This action cannot be undone!")
			fmt.Print("\nType the task ID to confirm: ")

			var confirmation string
			fmt.Scanln(&confirmation)

			if confirmation != taskID && confirmation != foundTask.ShortID {
				printer.Info("Deletion cancelled")
				return nil
			}
		}

		// TODO: Implement task deletion use case
		printer.Error("Task deletion not yet implemented")
		printer.Info("Would delete: %s - %s", foundTask.ShortID, foundTask.Title)

		return fmt.Errorf("task deletion not yet implemented")
	},
}

// taskCheckoutCmd checks out a git branch for a task
var taskCheckoutCmd = &cobra.Command{
	Use:   "checkout <task-id>",
	Short: "Checkout git branch for task",
	Long: `Checkout or create a git branch for a task.

The branch name is generated from the task ID and title using a configurable format.

Default branch format: {id}
Available placeholders:
  {id}       - Full task ID (e.g., TASK-123-fix-bug)
  {short-id} - Short ID (e.g., TASK-123)
  {slug}     - Title slug (e.g., fix-bug)

Examples:
  # Checkout branch with default format
  mkanban task checkout TASK-123

  # Checkout with custom format
  mkanban task checkout TASK-123 --branch-format "feature/{short-id}-{slug}"

  # Create new branch if it doesn't exist
  mkanban task checkout TASK-123 --create`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		// Get flags
		branchFormat, _ := cmd.Flags().GetString("branch-format")
		if branchFormat == "" {
			branchFormat = "{id}" // Default format
		}

		// Execute checkout use case
		err = container.CheckoutTaskUseCase.Execute(ctx, boardID, taskID, branchFormat)
		if err != nil {
			return fmt.Errorf("failed to checkout task: %w", err)
		}

		printer.Success("Checked out branch for task %s", taskID)
		return nil
	},
}

var taskCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current in-progress task(s)",
	Long: `Show tasks currently in the "In Progress" column.

Returns the task(s) you're currently working on.

Examples:
  # Show current task
  mkanban task current

  # Output for fzf
  mkanban task current -o fzf

  # JSON output
  mkanban task current -o json`,
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

		var inProgressTasks []dto.TaskDTO
		for _, col := range board.Columns {
			if col.Name == "In Progress" {
				inProgressTasks = col.Tasks
				break
			}
		}

		if len(inProgressTasks) == 0 {
			if outputFormat == "json" || outputFormat == "yaml" {
				return formatter.Print([]dto.TaskDTO{})
			}
			printer.Info("No tasks in progress")
			return nil
		}

		switch outputFormat {
		case "fzf":
			for _, task := range inProgressTasks {
				fmt.Printf("%s\t%s\n", task.ShortID, task.Title)
			}
		case "json", "yaml":
			return formatter.Print(inProgressTasks)
		default:
			if len(inProgressTasks) == 1 {
				task := inProgressTasks[0]
				printer.Println("%s %s", task.ShortID, task.Title)
			} else {
				printer.Header("In Progress (%d tasks)", len(inProgressTasks))
				fmt.Println()
				for _, task := range inProgressTasks {
					printer.Println("  %s %s", task.ShortID, task.Title)
				}
			}
		}

		return nil
	},
}

// taskShowCmd opens a task in the editor
var taskShowCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Open task in editor",
	Long: `Open a task file in your configured editor ($EDITOR).

Examples:
  # Open task for editing
  mkanban task show TASK-123`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		resolvedArgs, err := resolveArgs(args, 1)
		if err != nil {
			return err
		}
		taskID := resolvedArgs[0]

		boardID, err := getBoardID(ctx)
		if err != nil {
			return err
		}

		tasks, err := container.ListTasksUseCase.Execute(ctx, boardID)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		var foundTask *dto.TaskDTO
		for i, task := range tasks {
			if task.ShortID == taskID || task.ID == taskID {
				foundTask = &tasks[i]
				break
			}
		}

		if foundTask == nil {
			return fmt.Errorf("task '%s' not found", taskID)
		}

		if foundTask.FilePath == "" {
			return fmt.Errorf("task '%s' has no file path", taskID)
		}

		if _, err := os.Stat(foundTask.FilePath); err != nil {
			return fmt.Errorf("task file not accessible: %w", err)
		}

		return openEditorForTaskFile(foundTask.FilePath)
	},
}

// Helper functions

// openEditorForTask opens an editor for creating/editing task content
func openEditorForTask(title, description string) (string, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "mkanban-task-*.md")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write initial content
	content := fmt.Sprintf("# %s\n\n%s", title, description)
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	// Get editor from environment or use default
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Open editor
	cmd := exec.Command(editor, tmpPath)
	cleanup, err := attachEditorIO(cmd)
	if err != nil {
		return "", err
	}
	defer cleanup()

	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Read the edited file
	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}

	return string(edited), nil
}

func openEditorForEmptyTask(priority string, tags []string) (string, error) {
	tmpDir := filepath.Join(os.TempDir(), "mkanban")
	tmpPath := filepath.Join(tmpDir, "task")

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	if priority == "" {
		priority = "none"
	}

	if tags == nil {
		tags = []string{}
	}

	frontmatter := map[string]interface{}{
		"priority": priority,
		"tags":     tags,
	}

	content := "# "
	data, err := serialization.SerializeFrontmatter(frontmatter, content)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return "", err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	titleLine := findFirstHeadingLine(string(data))
	cmd := buildEditorCommand(editor, tmpPath, titleLine)
	cleanup, err := attachEditorIO(cmd)
	if err != nil {
		return "", err
	}
	defer cleanup()

	if err := cmd.Run(); err != nil {
		return "", err
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}

	return string(edited), nil
}

func openEditorForTaskFile(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	titleLine := findFirstHeadingLine(string(content))
	cmd := buildEditorCommand(editor, path, titleLine)
	cleanup, err := attachEditorIO(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	return cmd.Run()
}

func attachEditorIO(cmd *exec.Cmd) (func(), error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return func() {}, nil
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("no interactive terminal available (run without piping)")
	}

	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty

	return func() {
		_ = tty.Close()
	}, nil
}

func buildEditorCommand(editor, path string, line int) *exec.Cmd {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return exec.Command("vi", path)
	}

	editorName := filepath.Base(parts[0])
	if line > 0 && isViStyleEditor(editorName) {
		parts = append(parts, fmt.Sprintf("+%d", line))
	}
	parts = append(parts, path)

	return exec.Command(parts[0], parts[1:]...)
}

func isViStyleEditor(editor string) bool {
	switch editor {
	case "vi", "vim", "nvim":
		return true
	default:
		return false
	}
}

func findFirstHeadingLine(content string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return i + 1
		}
	}
	return 0
}

func parseTagsString(tagsStr string) []string {
	if tagsStr == "" {
		return nil
	}

	tags := strings.Split(tagsStr, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	filtered := tags[:0]
	for _, tag := range tags {
		if tag != "" {
			filtered = append(filtered, tag)
		}
	}

	return filtered
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
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "-", ""), " ", ""))
	return normalized == "todo"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	rootCmd.AddCommand(taskCmd)

	// Add subcommands
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskGetCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskMoveCmd)
	taskCmd.AddCommand(taskAdvanceCmd)
	taskCmd.AddCommand(taskRetreatCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskCheckoutCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskCurrentCmd)

	// taskListCmd flags
	taskListCmd.Flags().String("column", "", "Filter by column name")
	taskListCmd.Flags().String("priority", "", "Filter by priority (low, medium, high, critical)")
	taskListCmd.Flags().String("status", "", "Filter by status")
	taskListCmd.Flags().String("tag", "", "Filter by tag")
	taskListCmd.Flags().Bool("overdue", false, "Show only overdue tasks")
	taskListCmd.Flags().String("due-before", "", "Show tasks due before date (YYYY-MM-DD)")
	taskListCmd.Flags().Bool("all-boards", false, "List tasks from all boards")

	// taskCreateCmd flags
	taskCreateCmd.Flags().String("title", "", "Task title (optional; opens editor if omitted)")
	taskCreateCmd.Flags().String("description", "", "Task description")
	taskCreateCmd.Flags().String("column", "", "Column name (default: Todo)")
	taskCreateCmd.Flags().String("priority", "", "Priority: low, medium, high, critical")
	taskCreateCmd.Flags().String("status", "", "Status")
	taskCreateCmd.Flags().String("tags", "", "Comma-separated tags")
	taskCreateCmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	taskCreateCmd.Flags().Bool("edit", false, "Open editor for description")

	// taskUpdateCmd flags
	taskUpdateCmd.Flags().String("title", "", "New title")
	taskUpdateCmd.Flags().String("description", "", "New description")
	taskUpdateCmd.Flags().String("priority", "", "Priority: low, medium, high, critical")
	taskUpdateCmd.Flags().String("status", "", "Status")
	taskUpdateCmd.Flags().String("tags", "", "Set tags (replaces all existing)")
	taskUpdateCmd.Flags().StringSlice("add-tag", []string{}, "Add tag")
	taskUpdateCmd.Flags().StringSlice("remove-tag", []string{}, "Remove tag")
	taskUpdateCmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	taskUpdateCmd.Flags().Bool("edit", false, "Open editor for description")

	// taskDeleteCmd flags
	taskDeleteCmd.Flags().Bool("force", false, "Delete without confirmation")

	// taskCheckoutCmd flags
	taskCheckoutCmd.Flags().String("branch-format", "", "Branch name format (default: {id})")
	taskCheckoutCmd.Flags().Bool("create", false, "Create branch if it doesn't exist")
}
