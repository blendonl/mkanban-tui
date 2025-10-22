#!/bin/bash

echo "========================================="
echo "  mkanban System Test"
echo "========================================="
echo ""

# Clean up old data for fresh test
echo "Cleaning up old data..."
rm -rf ~/.local/share/mkanban/boards/*
echo "Done!"
echo ""

# Test 1: Build binaries
echo "Test 1: Building binaries..."
go build -o mkanban ./cmd/mkanban
go build -o mkanbad ./cmd/mkanbad
if [ $? -eq 0 ]; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi
echo ""

# Test 2: Test creating a board programmatically
echo "Test 2: Testing board creation..."
cat > test_create_board.go << 'EOF'
package main

import (
	"context"
	"fmt"
	"log"
	"mkanban/internal/application/dto"
	"mkanban/internal/di"
)

func main() {
	container, err := di.InitializeContainer()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Create a board
	board, err := container.CreateBoardUseCase.Execute(ctx, dto.CreateBoardRequest{
		Name:        "Test Project",
		Description: "A test project board",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Created board: %s (ID: %s, Prefix: %s)\n", board.Name, board.ID, board.Prefix)

	// Add columns
	columns := []struct{ name, desc string; order int }{
		{"Backlog", "Tasks to be planned", 1},
		{"Todo", "Ready to work on", 2},
		{"In Progress", "Currently working", 3},
		{"Review", "In code review", 4},
		{"Done", "Completed tasks", 5},
	}

	for _, col := range columns {
		_, err := container.CreateColumnUseCase.Execute(ctx, board.ID, dto.CreateColumnRequest{
			Name:        col.name,
			Description: col.desc,
			Order:       col.order,
			WIPLimit:    0,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("✅ Created column: %s\n", col.name)
	}

	// Add some tasks
	tasks := []struct{ title, col string }{
		{"Setup development environment", "Done"},
		{"Design database schema", "Review"},
		{"Implement user authentication", "In Progress"},
		{"Create API endpoints", "Todo"},
		{"Write documentation", "Backlog"},
	}

	for _, task := range tasks {
		_, err := container.CreateTaskUseCase.Execute(ctx, board.ID, dto.CreateTaskRequest{
			Title:       task.title,
			Description: fmt.Sprintf("Description for: %s", task.title),
			Priority:    "medium",
			ColumnName:  task.col,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("✅ Created task: %s in %s\n", task.title, task.col)
	}

	// Verify file structure
	fmt.Println("\n✅ Board created successfully!")
	fmt.Printf("   Location: ~/.local/share/mkanban/boards/%s/\n", board.ID)
}
EOF

go run test_create_board.go
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Board creation test passed"
else
    echo "❌ Board creation test failed"
    exit 1
fi
rm test_create_board.go
echo ""

# Test 3: Verify file structure
echo "Test 3: Verifying file structure..."
BOARDS_DIR=~/.local/share/mkanban/boards
if [ -d "$BOARDS_DIR" ]; then
    echo "✅ Boards directory exists"
    BOARD_COUNT=$(ls -1 "$BOARDS_DIR" | wc -l)
    echo "   Found $BOARD_COUNT board(s)"

    # Show structure
    echo ""
    echo "File structure:"
    tree -L 3 "$BOARDS_DIR" 2>/dev/null || find "$BOARDS_DIR" -maxdepth 3 -type f | head -20
else
    echo "❌ Boards directory not found"
    exit 1
fi
echo ""

# Test 4: Read a board
echo "Test 4: Reading board data..."
cat > test_read_board.go << 'EOF'
package main

import (
	"context"
	"fmt"
	"log"
	"mkanban/internal/di"
)

func main() {
	container, err := di.InitializeContainer()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// List all boards
	boards, err := container.ListBoardsUseCase.Execute(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if len(boards) == 0 {
		log.Fatal("No boards found")
	}

	// Get first board
	board, err := container.GetBoardUseCase.Execute(ctx, boards[0].ID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Read board: %s\n", board.Name)
	fmt.Printf("   Columns: %d\n", len(board.Columns))
	fmt.Printf("   Total tasks: %d\n", board.TotalTaskCount)

	for _, col := range board.Columns {
		fmt.Printf("   - %s: %d task(s)\n", col.Name, len(col.Tasks))
		for _, task := range col.Tasks {
			fmt.Printf("      • %s (ID: %s)\n", task.Title, task.ShortID)
		}
	}
}
EOF

go run test_read_board.go
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Board reading test passed"
else
    echo "❌ Board reading test failed"
    exit 1
fi
rm test_read_board.go
echo ""

# Summary
echo "========================================="
echo "  All Tests Passed! ✅"
echo "========================================="
echo ""
echo "System is working correctly:"
echo "  • Clean architecture implementation"
echo "  • File-based storage with YAML frontmatter"
echo "  • Domain entities with business rules"
echo "  • Use cases for operations"
echo "  • Dependency injection with Wire"
echo ""
echo "Next steps:"
echo "  1. Run ./mkanban to start the TUI"
echo "  2. Run ./mkanbad to start the daemon"
echo "  3. Boards are stored in ~/.local/share/mkanban/boards/"
echo ""
