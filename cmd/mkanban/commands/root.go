package commands

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"mkanban/cmd/mkanban/output"
	"mkanban/internal/daemon"
	"mkanban/internal/di"
	"mkanban/internal/infrastructure/config"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"

	// Global flags
	boardID      string
	outputFormat string
	configPath   string
	quiet        bool

	// Shared instances
	cfg       *config.Config
	container *di.Container
	printer   *output.Printer
	formatter *output.Formatter
)

var multiSpaceRE = regexp.MustCompile(`\s{2,}`)
var taskIDLikeRE = regexp.MustCompile(`^[A-Za-z]+-\d+`)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mkanban",
	Short: "Terminal-based Kanban board with git integration",
	Long: `mkanban is a powerful terminal-based Kanban board system with git workflow integration.

Features:
  - Multiple boards with customizable columns
  - Task management with priorities, tags, and due dates
  - Git integration for branch-per-task workflows
  - Automated actions and reminders
  - Tmux session awareness
  - Interactive TUI and comprehensive CLI

Examples:
  # Launch interactive TUI
  mkanban
  mkanban tui

  # List all boards
  mkanban board list

  # Create a new task
  mkanban task create --title "Fix login bug" --priority high

  # List tasks in a specific column
  mkanban task list --column "In Progress"

  # Move task to next column
  mkanban task advance TASK-123

  # Checkout git branch for task
  mkanban task checkout TASK-123`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration
		var err error
		loader, err := config.NewLoader()
		if err != nil {
			return fmt.Errorf("failed to create config loader: %w", err)
		}

		// Load config (custom path support via --config flag would require LoadFrom method)
		cfg, err = loader.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// TODO: Add support for custom config path when LoadFrom is implemented
		if configPath != "" {
			return fmt.Errorf("custom config path not yet supported. Config file: %s", loader.GetConfigPath())
		}

		// Initialize DI container
		container, err = di.InitializeContainer()
		if err != nil {
			return fmt.Errorf("failed to initialize container: %w", err)
		}

		// Initialize output formatter
		format, err := output.ParseFormat(outputFormat)
		if err != nil {
			return err
		}
		formatter = output.NewFormatter(format, os.Stdout)
		printer = output.DefaultPrinter()

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		printer := output.DefaultPrinter()
		printer.Error("%v", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&boardID, "board-id", "b", "", "Board to operate on (default: active board from session)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, yaml")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")

	// Version flag
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")

	// Handle version flag
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			printVersion()
			return
		}

		// Default behavior: show help or launch TUI
		if len(args) == 0 {
			// Launch TUI
			if err := tuiCmd.RunE(cmd, args); err != nil {
				printer.Error("%v", err)
				os.Exit(1)
			}
		} else {
			cmd.Help()
		}
	}
}

// printVersion prints version information
func printVersion() {
	fmt.Printf("mkanban version %s\n", Version)
	fmt.Printf("  Git commit: %s\n", GitCommit)
	fmt.Printf("  Built:      %s\n", BuildDate)
}

// getContext returns a context for command execution
func getContext() context.Context {
	return context.Background()
}

// getBoardID returns the board ID to use for commands
// Priority: flag > active session > first board
func getBoardID(ctx context.Context) (string, error) {
	// If board ID is specified via flag, use it
	if boardID != "" {
		return boardID, nil
	}

	// Try to get active board from session/daemon
	if activeBoardID, err := getActiveBoardFromSession(ctx); err == nil && activeBoardID != "" {
		return activeBoardID, nil
	}

	// Fall back to first board
	boards, err := container.ListBoardsUseCase.Execute(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list boards: %w", err)
	}

	if len(boards) == 0 {
		return "", fmt.Errorf("no boards found. Create a board first with: mkanban board create <name>")
	}

	return boards[0].ID, nil
}

func resolveArgs(args []string, expected int) ([]string, error) {
	if len(args) >= expected {
		return args, nil
	}

	pipedArgs, err := readPipedArgs(expected)
	if err != nil {
		return nil, err
	}

	needed := expected - len(args)
	available := len(args) + len(pipedArgs)
	if len(pipedArgs) < needed {
		return nil, fmt.Errorf("accepts %d arg(s), received %d", expected, available)
	}

	resolved := append([]string{}, pipedArgs[:needed]...)
	resolved = append(resolved, args...)
	return resolved, nil
}

func readPipedArgs(expected int) ([]string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	return extractArgsFromInput(data, expected), nil
}

func extractArgsFromInput(data []byte, expected int) []string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	bestScore := -1
	var best []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		tokens, score := parsePipedLine(line, expected)
		if len(tokens) < expected {
			continue
		}

		if score > bestScore {
			bestScore = score
			best = tokens
		}
	}

	if len(best) == 0 {
		return nil
	}
	return best
}

func parsePipedLine(line string, expected int) ([]string, int) {
	if strings.Contains(line, "\t") {
		return splitFields(line, func(r rune) bool { return r == '\t' }), 3
	}
	if strings.Contains(line, " :: ") {
		parts := strings.Split(line, " :: ")
		return parts, 3
	}
	if multiSpaceRE.MatchString(line) {
		return multiSpaceRE.Split(line, -1), 2
	}

	fields := strings.Fields(line)
	if expected == 1 && len(fields) > 1 {
		if taskIDLikeRE.MatchString(fields[0]) {
			return []string{fields[0]}, 2
		}
		return []string{line}, 1
	}

	return fields, 1
}

func splitFields(input string, split func(rune) bool) []string {
	fields := strings.FieldsFunc(input, split)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

// getActiveBoardFromSession attempts to get the active board ID from the current session
func getActiveBoardFromSession(ctx context.Context) (string, error) {
	// Check if running in tmux
	if os.Getenv("TMUX") == "" {
		return "", fmt.Errorf("not in tmux session")
	}

	// Create daemon client
	client := daemon.NewClient(cfg)

	// Connect to daemon
	if err := client.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	// Get active board from daemon
	boardID, err := client.GetActiveBoard(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get active board: %w", err)
	}

	return boardID, nil
}
