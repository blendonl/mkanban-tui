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
	Use:   "mnotes",
	Short: "Note-taking system for mkanban",
	Long: `mnotes is a note-taking system integrated with mkanban.

It supports:
- Daily journal entries
- Meeting notes with automatic task linking
- Global and project-specific notes
- Full-text search across notes

Examples:
  # Create today's journal entry
  mnotes journal

  # Create a new note
  mnotes new "Meeting with team"

  # List today's notes
  mnotes list --today

  # Search notes
  mnotes search "api design"`,
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
