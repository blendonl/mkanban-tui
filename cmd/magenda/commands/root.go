package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"mkanban/internal/di"
)

var container *di.Container

var rootCmd = &cobra.Command{
	Use:   "magenda",
	Short: "Agenda and scheduling system for mkanban",
	Long: `magenda is an agenda system integrated with mkanban.

It provides:
- Daily and weekly views of scheduled tasks
- Meeting management with Google Calendar sync
- Time blocking and scheduling
- Recurring task support

Examples:
  # Show today's agenda
  magenda today

  # Show this week's agenda
  magenda week

  # Schedule a task
  magenda schedule TASK-123 --date 2025-01-15 --time 10:00

  # Create a meeting
  magenda meeting "Sprint Planning" --date 2025-01-15 --time 14:00 --duration 1h`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		container, err = di.InitializeContainer()
		if err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getContext() context.Context {
	return context.Background()
}

func init() {
	rootCmd.PersistentFlags().StringP("project", "p", "", "Project ID or slug")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text, json, yaml)")
}
