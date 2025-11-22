package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data to new formats",
	Long: `Migrate boards and tasks to new data formats.

This command is used when upgrading mkanban to handle schema changes and data format updates.

Examples:
  # Migrate all boards to new format
  mkanban migrate

  # Migrate specific board
  mkanban migrate --board-id my-project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()

		if !quiet {
			printer.Info("Starting column migration to new format...")
		}

		// Get all boards or specific board
		var boardsToMigrate []struct {
			ID   string
			Name string
		}

		if boardID != "" {
			// Migrate specific board
			board, err := container.GetBoardUseCase.Execute(ctx, boardID)
			if err != nil {
				return fmt.Errorf("failed to get board: %w", err)
			}
			boardsToMigrate = append(boardsToMigrate, struct {
				ID   string
				Name string
			}{ID: board.ID, Name: board.Name})
		} else {
			// Migrate all boards
			boards, err := container.ListBoardsUseCase.Execute(ctx)
			if err != nil {
				return fmt.Errorf("failed to list boards: %w", err)
			}

			if len(boards) == 0 {
				printer.Info("No boards found. Nothing to migrate.")
				return nil
			}

			for _, b := range boards {
				boardsToMigrate = append(boardsToMigrate, struct {
					ID   string
					Name string
				}{ID: b.ID, Name: b.Name})
			}
		}

		// Migrate each board
		migratedCount := 0
		for _, board := range boardsToMigrate {
			if !quiet {
				printer.Info("Migrating board: %s (%s)", board.Name, board.ID)
			}

			// Get the board repository from the container
			boardRepo := container.BoardRepo

			// Cast to filesystem implementation to access migration method
			if fsRepo, ok := boardRepo.(interface {
				MigrateColumnsToNewFormat(ctx interface{}, boardID string) error
			}); ok {
				err := fsRepo.MigrateColumnsToNewFormat(ctx, board.ID)
				if err != nil {
					printer.Error("Error migrating board %s: %v", board.ID, err)
				} else {
					printer.Success("Board %s migrated successfully", board.ID)
					migratedCount++
				}
			} else {
				printer.Warning("Board repository does not support migration")
			}
		}

		if !quiet {
			fmt.Println()
			printer.Success("Migration complete. Migrated %d board(s).", migratedCount)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
