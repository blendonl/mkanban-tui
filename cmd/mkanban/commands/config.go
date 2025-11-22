package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"mkanban/internal/infrastructure/config"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage mkanban configuration settings.

Configuration is stored in YAML format at:
  ~/.config/mkanban/config.yml

Examples:
  # Show current configuration
  mkanban config show

  # Get a specific config value
  mkanban config get storage.base_path

  # Set a config value
  mkanban config set tui.theme.primary_color "#FF5733"

  # Edit config in editor
  mkanban config edit

  # Show config file location
  mkanban config path

  # Reset config to defaults
  mkanban config reset`,
}

// configShowCmd shows the current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Show the current configuration settings.

Examples:
  # Show in YAML format (default)
  mkanban config show

  # Show in JSON format
  mkanban config show --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch outputFormat {
		case "json", "yaml":
			return formatter.Print(cfg)
		default:
			// Print as YAML (default)
			return formatter.Print(cfg)
		}
	},
}

// configGetCmd gets a specific config value
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Long: `Get a specific configuration value by key.

Use dot notation for nested values.

Examples:
  # Get storage path
  mkanban config get storage.base_path

  # Get theme color
  mkanban config get tui.theme.primary_color`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// TODO: Implement config key lookup
		printer.Error("Config get not yet implemented")
		printer.Info("Would get value for key: %s", key)
		printer.Info("Use 'mkanban config show' to see all values")

		return fmt.Errorf("config get not yet implemented")
	},
}

// configSetCmd sets a config value
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long: `Set a configuration value.

Use dot notation for nested values.

WARNING: This modifies your config file. Make a backup first if unsure.

Examples:
  # Set storage path
  mkanban config set storage.base_path "/custom/path"

  # Set theme color
  mkanban config set tui.theme.primary_color "#FF5733"

  # Enable session tracking
  mkanban config set session.enabled true`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// TODO: Implement config update
		printer.Error("Config set not yet implemented")
		printer.Info("Would set %s = %s", key, value)
		printer.Info("Use 'mkanban config edit' to manually edit the config file")

		return fmt.Errorf("config set not yet implemented")
	},
}

// configEditCmd opens the config file in an editor
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit config in editor",
	Long: `Open the configuration file in your default editor.

The editor is determined by the EDITOR environment variable (default: vi).

Examples:
  # Edit config with default editor
  mkanban config edit

  # Edit with specific editor
  EDITOR=nano mkanban config edit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get config path
		loader, err := config.NewLoader()
		if err != nil {
			return fmt.Errorf("failed to create config loader: %w", err)
		}

		configPath := loader.GetConfigPath()

		// Get editor from environment or use default
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		printer.Info("Opening config file: %s", configPath)
		printer.Subtle("Editor: %s", editor)

		// Open editor
		editorCmd := exec.Command(editor, configPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("failed to run editor: %w", err)
		}

		printer.Success("Config file edited")
		printer.Info("Restart mkanban for changes to take effect")

		return nil
	},
}

// configPathCmd shows the config file path
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file location",
	Long: `Show the path to the configuration file.

Examples:
  # Show config path
  mkanban config path`,
	RunE: func(cmd *cobra.Command, args []string) error {
		loader, err := config.NewLoader()
		if err != nil {
			return fmt.Errorf("failed to create config loader: %w", err)
		}

		configPath := loader.GetConfigPath()
		fmt.Println(configPath)

		return nil
	},
}

// configResetCmd resets the config to defaults
var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset config to defaults",
	Long: `Reset the configuration to default values.

WARNING: This will overwrite your current configuration.
Make a backup first if you want to preserve custom settings.

Examples:
  # Reset config (with confirmation)
  mkanban config reset

  # Reset without confirmation
  mkanban config reset --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		loader, err := config.NewLoader()
		if err != nil {
			return fmt.Errorf("failed to create config loader: %w", err)
		}

		configPath := loader.GetConfigPath()

		// Confirm reset unless --force is used
		if !force {
			printer.Warning("About to reset configuration to defaults")
			printer.Warning("Current config: %s", configPath)
			printer.Warning("This action cannot be undone!")
			fmt.Print("\nType 'yes' to confirm: ")

			var confirmation string
			fmt.Scanln(&confirmation)

			if confirmation != "yes" {
				printer.Info("Reset cancelled")
				return nil
			}
		}

		// TODO: Implement config reset
		printer.Error("Config reset not yet implemented")
		printer.Info("Would reset config file: %s", configPath)
		printer.Info("For now, manually delete the file and it will be recreated with defaults")

		return fmt.Errorf("config reset not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configResetCmd)

	// configResetCmd flags
	configResetCmd.Flags().Bool("force", false, "Reset without confirmation")
}
