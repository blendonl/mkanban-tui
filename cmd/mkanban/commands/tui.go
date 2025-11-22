package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"mkanban/internal/daemon"
	"mkanban/tui"
	"mkanban/tui/style"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui [board-id]",
	Short: "Launch interactive terminal user interface",
	Long: `Launch the interactive TUI (Terminal User Interface) for managing your Kanban board.

The TUI provides a visual, keyboard-driven interface for:
  - Viewing tasks across columns
  - Creating and editing tasks
  - Moving tasks between columns
  - Managing task priorities and due dates

Keyboard shortcuts:
  ←/h      - Move to left column
  →/l      - Move to right column
  ↑/k      - Move to task above
  ↓/j      - Move to task below
  a        - Add new task to current column
  m/Enter  - Move task to next column
  d        - Delete selected task
  q/Ctrl+C - Quit application

Examples:
  # Launch TUI with default board
  mkanban tui

  # Launch TUI with specific board
  mkanban tui --board-id my-project

  # Launch TUI (shorthand - default command)
  mkanban`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()

		// Initialize styles and keybindings from config
		style.InitStyles(cfg)
		tui.InitKeybindings(cfg)

		// Determine which board to use
		var selectedBoardID string
		var err error

		// Check if board ID is provided as positional argument
		if len(args) > 0 {
			selectedBoardID = args[0]
		} else {
			selectedBoardID, err = getBoardID(ctx)
			if err != nil {
				return err
			}
		}

		// Create daemon client
		daemonClient := daemon.NewClient(cfg)

		// Connect to daemon (will auto-start if needed)
		if err := daemonClient.Connect(); err != nil {
			return fmt.Errorf("failed to connect to daemon: %w", err)
		}
		defer daemonClient.Close()

		// Load the board via daemon
		boardDTO, err := daemonClient.GetBoard(ctx, selectedBoardID)
		if err != nil {
			return fmt.Errorf("failed to load board: %w", err)
		}

		if !quiet {
			printer.Info("Launching TUI for board: %s", boardDTO.Name)
		}

		// Create TUI model with daemon client
		m := tui.NewModel(boardDTO, daemonClient, cfg, selectedBoardID)

		// Start the program
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}

		// Cleanup subscription
		daemonClient.Unsubscribe()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
