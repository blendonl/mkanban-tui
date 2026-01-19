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
	"mkanban/internal/di"
)

var container *di.Container
var taskIDLikeRE = regexp.MustCompile(`^[A-Za-z]+-\d+`)

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
	if strings.Contains(line, "  ") {
		return strings.Fields(line), 2
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

func init() {
	rootCmd.PersistentFlags().StringP("project", "p", "", "Project ID or slug")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text, json, yaml)")
}
