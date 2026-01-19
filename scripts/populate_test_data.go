package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"mkanban/internal/application/dto"
	"mkanban/internal/di"
	"mkanban/internal/infrastructure/config"
	"mkanban/pkg/slug"
)

func main() {
	// Load configuration
	loader, err := config.NewLoader()
	if err != nil {
		log.Fatalf("Failed to create config loader: %v", err)
	}

	_, err = loader.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize DI container
	container, err := di.InitializeContainer()
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}

	ctx := context.Background()

	// Get or create board
	boards, err := container.ListBoardsUseCase.Execute(ctx)
	if err != nil {
		log.Fatalf("Failed to list boards: %v", err)
	}

	var boardID string
	if len(boards) == 0 {
		// Create default board
		board, err := container.CreateBoardUseCase.Execute(ctx, struct {
			ProjectID   string `json:"project_id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		}{
			ProjectID:   slug.Generate("Test Project"),
			Name:        "Test Board",
			Description: "Board with test data",
		})
		if err != nil {
			log.Fatalf("Failed to create board: %v", err)
		}
		boardID = board.ID

		// Add default columns
		columns := []struct {
			name  string
			order int
		}{
			{"Todo", 1},
			{"In Progress", 2},
			{"Done", 3},
		}

		for _, col := range columns {
			_, err := container.CreateColumnUseCase.Execute(ctx, boardID, struct {
				Name        string  `json:"name"`
				Description string  `json:"description"`
				Order       int     `json:"order"`
				WIPLimit    int     `json:"wip_limit"`
				Color       *string `json:"color,omitempty"`
			}{
				Name:     col.name,
				Order:    col.order,
				WIPLimit: 0,
			})
			if err != nil {
				log.Fatalf("Failed to create column: %v", err)
			}
		}
	} else {
		boardID = boards[0].ID
	}

	// Create sample tasks with various properties
	testTasks := []struct {
		title       string
		description string
		priority    string
		column      string
		tags        []string
		daysOffset  *int // nil = no due date, negative = overdue, positive = future
	}{
		{
			title:       "Fix authentication bug",
			description: "Users are unable to log in with OAuth. Need to debug the callback handler.",
			priority:    "high",
			column:      "In Progress",
			tags:        []string{"bug", "auth", "urgent"},
			daysOffset:  intPtr(-2), // Overdue by 2 days
		},
		{
			title:       "Implement user dashboard",
			description: "Create a dashboard page showing user stats and recent activity",
			priority:    "medium",
			column:      "In Progress",
			tags:        []string{"feature", "frontend"},
			daysOffset:  intPtr(5), // Due in 5 days
		},
		{
			title:       "Write documentation for API",
			description: "Document all REST endpoints with examples and response formats",
			priority:    "low",
			column:      "In Progress",
			tags:        []string{"docs", "api"},
			daysOffset:  intPtr(10),
		},
		{
			title:       "Add dark mode support",
			description: "Implement theme switching between light and dark modes",
			priority:    "medium",
			column:      "Done",
			tags:        []string{"feature", "ui", "enhancement"},
			daysOffset:  intPtr(1), // Due tomorrow
		},
		{
			title:       "Optimize database queries",
			description: "Profile and optimize slow queries in the user service",
			priority:    "high",
			column:      "In Progress",
			tags:        []string{"performance", "backend"},
			daysOffset:  intPtr(0), // Due today
		},
		{
			title:       "Setup CI/CD pipeline",
			description: "Configure GitHub Actions for automated testing and deployment",
			priority:    "",
			column:      "In Progress",
			tags:        []string{"devops", "automation"},
			daysOffset:  nil, // No due date
		},
		{
			title:       "Refactor authentication module",
			description: "Clean up the auth code and improve error handling",
			priority:    "low",
			column:      "Done",
			tags:        []string{"refactor", "auth"},
			daysOffset:  nil,
		},
		{
			title:       "Update dependencies",
			description: "Update all npm packages to latest versions and test for breaking changes",
			priority:    "medium",
			column:      "Done",
			tags:        []string{"maintenance"},
			daysOffset:  intPtr(7),
		},
	}

	fmt.Println("Creating sample tasks...")

	for _, task := range testTasks {
		var dueDate *time.Time
		if task.daysOffset != nil {
			date := time.Now().AddDate(0, 0, *task.daysOffset)
			dueDate = &date
		}

		taskDTO, err := container.CreateTaskUseCase.Execute(ctx, boardID, dto.CreateTaskRequest{
			Title:       task.title,
			Description: task.description,
			Priority:    task.priority,
			ColumnName:  task.column,
			Tags:        task.tags,
			DueDate:     dueDate,
		})
		if err != nil {
			log.Printf("Failed to create task '%s': %v", task.title, err)
			continue
		}

		fmt.Printf("✓ Created: %s [%s] in %s\n", taskDTO.Title, task.priority, task.column)
	}

	fmt.Println("\nSample data created successfully!")
	fmt.Println("Run './mkanban' to view the board with the new design.")
}

func intPtr(i int) *int {
	return &i
}
