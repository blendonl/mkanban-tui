package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigFileName = "config.yml"
	defaultConfigDirName  = ".config/mkanban"
	defaultBoardsDirName  = "boards"
	defaultDataDirName    = ".local/share/mkanban"
)

// Config holds application configuration
type Config struct {
	Storage        StorageConfig        `yaml:"storage"`
	Daemon         DaemonConfig         `yaml:"daemon"`
	TUI            TUIConfig            `yaml:"tui"`
	Keybindings    KeybindingsConfig    `yaml:"keybindings"`
	SessionTracking SessionTrackingConfig `yaml:"session_tracking"`
	Actions        ActionsConfig        `yaml:"actions"`
}

// StorageConfig holds storage-related configuration
type StorageConfig struct {
	BoardsPath string `yaml:"boards_path"`
	DataPath   string `yaml:"data_path"`
}

// DaemonConfig holds daemon-related configuration
type DaemonConfig struct {
	SocketDir  string `yaml:"socket_dir"`
	SocketName string `yaml:"socket_name"`
}

// TUIConfig holds TUI styling configuration
type TUIConfig struct {
	Styles StylesConfig `yaml:"styles"`
}

// StylesConfig holds color and styling configuration
type StylesConfig struct {
	Column            ColumnStyle       `yaml:"column"`
	FocusedColumn     ColumnStyle       `yaml:"focused_column"`
	ColumnTitle       TextStyle         `yaml:"column_title"`
	Task              TextStyle         `yaml:"task"`
	SelectedTask      TextStyle         `yaml:"selected_task"`
	Help              TextStyle         `yaml:"help"`
	TaskCard          TaskCardStyle     `yaml:"task_card"`
	SelectedTaskCard  TaskCardStyle     `yaml:"selected_task_card"`
	Description       TextStyle         `yaml:"description"`
	Tag               TextStyle         `yaml:"tag"`
	DueDate           TextStyle         `yaml:"due_date"`
	Overdue           TextStyle         `yaml:"overdue"`
	Priority          PriorityColors    `yaml:"priority"`
	DueDateUrgency    DueDateColors     `yaml:"due_date_urgency"`
	ScrollIndicator   TextStyle         `yaml:"scroll_indicator"`
}

// ColumnStyle represents column styling
type ColumnStyle struct {
	PaddingVertical   int    `yaml:"padding_vertical"`
	PaddingHorizontal int    `yaml:"padding_horizontal"`
	BorderStyle       string `yaml:"border_style"`
	BorderColor       string `yaml:"border_color"`
}

// TextStyle represents text styling
type TextStyle struct {
	Foreground        string `yaml:"foreground,omitempty"`
	Background        string `yaml:"background,omitempty"`
	Bold              bool   `yaml:"bold,omitempty"`
	Italic            bool   `yaml:"italic,omitempty"`
	PaddingVertical   int    `yaml:"padding_vertical,omitempty"`
	PaddingHorizontal int    `yaml:"padding_horizontal,omitempty"`
	Align             string `yaml:"align,omitempty"`
}

// TaskCardStyle represents task card border styling
type TaskCardStyle struct {
	BorderColor string `yaml:"border_color"`
}

// PriorityColors holds colors for different priority levels
type PriorityColors struct {
	High    string `yaml:"high"`
	Medium  string `yaml:"medium"`
	Low     string `yaml:"low"`
	Default string `yaml:"default"`
}

// DueDateColors holds colors for different due date urgency levels
type DueDateColors struct {
	Overdue   string `yaml:"overdue"`
	DueSoon   string `yaml:"due_soon"`
	Upcoming  string `yaml:"upcoming"`
	FarFuture string `yaml:"far_future"`
}

// KeybindingsConfig holds keybinding configuration
type KeybindingsConfig struct {
	Up     []string `yaml:"up"`
	Down   []string `yaml:"down"`
	Left   []string `yaml:"left"`
	Right  []string `yaml:"right"`
	Move   []string `yaml:"move"`
	Add    []string `yaml:"add"`
	Delete []string `yaml:"delete"`
	Quit   []string `yaml:"quit"`
}

// SessionTrackingConfig holds session tracking configuration
type SessionTrackingConfig struct {
	Enabled          bool   `yaml:"enabled"`
	PollInterval     int    `yaml:"poll_interval"` // in seconds
	TrackerType      string `yaml:"tracker_type"`  // "tmux", "zellij", etc.
	GeneralBoardName string `yaml:"general_board_name"`
	GitSync          GitSyncConfig `yaml:"git_sync"`
}

// GitSyncConfig holds git synchronization configuration
type GitSyncConfig struct {
	Enabled            bool `yaml:"enabled"`
	AutoSyncBranches   bool `yaml:"auto_sync_branches"`
	WatchForChanges    bool `yaml:"watch_for_changes"`
	CreateTasksForRemotes bool `yaml:"create_tasks_for_remotes"`
}

// ActionsConfig holds actions/reminders configuration
type ActionsConfig struct {
	Enabled          bool                 `yaml:"enabled"`
	CheckInterval    int                  `yaml:"check_interval"` // in seconds, for time-based actions
	NotificationsEnabled bool             `yaml:"notifications_enabled"`
	ScriptsEnabled   bool                 `yaml:"scripts_enabled"`
	ScriptsDir       string               `yaml:"scripts_dir"`
	Templates        []ActionTemplate     `yaml:"templates"`
}

// ActionTemplate represents a reusable action template
type ActionTemplate struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Trigger     TriggerConfig          `yaml:"trigger"`
	ActionType  ActionTypeConfig       `yaml:"action_type"`
	Conditions  []ConditionConfig      `yaml:"conditions,omitempty"`
}

// TriggerConfig represents trigger configuration
type TriggerConfig struct {
	Type     string            `yaml:"type"` // "time" or "event"
	Schedule *ScheduleConfig   `yaml:"schedule,omitempty"`
	Event    string            `yaml:"event,omitempty"` // event type for event triggers
}

// ScheduleConfig represents schedule configuration
type ScheduleConfig struct {
	Type     string `yaml:"type"` // "absolute", "relative_due_date", "relative_creation", "recurring"
	Time     string `yaml:"time,omitempty"` // ISO 8601 format for absolute
	Offset   string `yaml:"offset,omitempty"` // duration string like "1h", "30m", "2d"
	CronExpr string `yaml:"cron,omitempty"` // cron expression for recurring
}

// ActionTypeConfig represents action type configuration
type ActionTypeConfig struct {
	Type           string            `yaml:"type"` // "notification", "script", "task_mutation", "task_movement", "task_creation"
	// For notifications
	Title          string            `yaml:"title,omitempty"`
	Message        string            `yaml:"message,omitempty"`
	// For scripts
	ScriptPath     string            `yaml:"script_path,omitempty"`
	ScriptEnv      map[string]string `yaml:"script_env,omitempty"`
	// For task mutations
	UpdatePriority string            `yaml:"update_priority,omitempty"`
	UpdateStatus   string            `yaml:"update_status,omitempty"`
	AddTags        []string          `yaml:"add_tags,omitempty"`
	RemoveTags     []string          `yaml:"remove_tags,omitempty"`
	SetMetadata    map[string]string `yaml:"set_metadata,omitempty"`
	// For task movement
	TargetColumn   string            `yaml:"target_column,omitempty"`
	// For task creation
	TaskTitle      string            `yaml:"task_title,omitempty"`
	TaskDescription string           `yaml:"task_description,omitempty"`
	TaskPriority   string            `yaml:"task_priority,omitempty"`
	TaskStatus     string            `yaml:"task_status,omitempty"`
	TaskColumn     string            `yaml:"task_column,omitempty"`
	TaskTags       []string          `yaml:"task_tags,omitempty"`
	TaskMetadata   map[string]string `yaml:"task_metadata,omitempty"`
}

// ConditionConfig represents condition configuration
type ConditionConfig struct {
	Field    string      `yaml:"field"`
	Operator string      `yaml:"operator"`
	Value    interface{} `yaml:"value"`
}

// Loader handles loading and saving configuration
type Loader struct {
	configPath string
}

// NewLoader creates a new config loader
func NewLoader() (*Loader, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, defaultConfigDirName)
	configPath := filepath.Join(configDir, defaultConfigFileName)

	return &Loader{
		configPath: configPath,
	}, nil
}

// Load loads the configuration, creating defaults if it doesn't exist
func (l *Loader) Load() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		// Create default config
		return l.createDefaultConfig()
	}

	// Read existing config
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save persists the configuration to disk
func (l *Loader) Save(config *Config) error {
	// Ensure config directory exists
	configDir := filepath.Dir(l.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(l.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// createDefaultConfig creates and saves a default configuration
func (l *Loader) createDefaultConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, defaultDataDirName)
	boardsPath := filepath.Join(dataDir, defaultBoardsDirName)
	socketDir := filepath.Join(homeDir, defaultDataDirName)

	config := &Config{
		Storage: StorageConfig{
			BoardsPath: boardsPath,
			DataPath:   dataDir,
		},
		Daemon: DaemonConfig{
			SocketDir:  socketDir,
			SocketName: "mkanbad.sock",
		},
		TUI: TUIConfig{
			Styles: StylesConfig{
				Column: ColumnStyle{
					PaddingVertical:   1,
					PaddingHorizontal: 2,
					BorderStyle:       "rounded",
					BorderColor:       "240",
				},
				FocusedColumn: ColumnStyle{
					PaddingVertical:   1,
					PaddingHorizontal: 2,
					BorderStyle:       "rounded",
					BorderColor:       "62",
				},
				ColumnTitle: TextStyle{
					Foreground: "99",
					Bold:       true,
					Align:      "center",
				},
				Task: TextStyle{
					Foreground:        "252",
					PaddingVertical:   0,
					PaddingHorizontal: 1,
				},
				SelectedTask: TextStyle{
					Foreground:        "230",
					Background:        "62",
					Bold:              true,
					PaddingVertical:   0,
					PaddingHorizontal: 1,
				},
				Help: TextStyle{
					Foreground:        "241",
					PaddingVertical:   1,
					PaddingHorizontal: 2,
				},
				TaskCard: TaskCardStyle{
					BorderColor: "#444444",
				},
				SelectedTaskCard: TaskCardStyle{
					BorderColor: "#A8DADC",
				},
				Description: TextStyle{
					Foreground:        "#888888",
					Italic:            true,
					PaddingHorizontal: 2,
				},
				Tag: TextStyle{
					Foreground:        "#A8DADC",
					PaddingHorizontal: 2,
				},
				DueDate: TextStyle{
					Foreground:        "#999999",
					PaddingHorizontal: 2,
				},
				Overdue: TextStyle{
					Foreground:        "#FF6B6B",
					Bold:              true,
					PaddingHorizontal: 2,
				},
				Priority: PriorityColors{
					High:    "#FF6B6B",
					Medium:  "#FFE66D",
					Low:     "#95E1D3",
					Default: "#999999",
				},
				DueDateUrgency: DueDateColors{
					Overdue:   "#FF6B6B",
					DueSoon:   "#FFE66D",
					Upcoming:  "#A8DADC",
					FarFuture: "#999999",
				},
				ScrollIndicator: TextStyle{
					Foreground: "#999999",
					Bold:       true,
				},
			},
		},
		Keybindings: KeybindingsConfig{
			Up:     []string{"up", "k"},
			Down:   []string{"down", "j"},
			Left:   []string{"left", "h"},
			Right:  []string{"right", "l"},
			Move:   []string{"m", "enter"},
			Add:    []string{"a"},
			Delete: []string{"d"},
			Quit:   []string{"q", "ctrl+c"},
		},
		SessionTracking: SessionTrackingConfig{
			Enabled:          true,
			PollInterval:     5,
			TrackerType:      "tmux",
			GeneralBoardName: "General Tasks",
			GitSync: GitSyncConfig{
				Enabled:               true,
				AutoSyncBranches:      true,
				WatchForChanges:       true,
				CreateTasksForRemotes: false,
			},
		},
		Actions: ActionsConfig{
			Enabled:              true,
			CheckInterval:        60, // Check every minute
			NotificationsEnabled: true,
			ScriptsEnabled:       true,
			ScriptsDir:           filepath.Join(homeDir, ".config", "mkanban", "scripts"),
			Templates: []ActionTemplate{
				{
					ID:          "due-tomorrow-reminder",
					Name:        "Due Tomorrow Reminder",
					Description: "Notify when a task is due tomorrow",
					Trigger: TriggerConfig{
						Type: "time",
						Schedule: &ScheduleConfig{
							Type:   "relative_due_date",
							Offset: "1d",
						},
					},
					ActionType: ActionTypeConfig{
						Type:    "notification",
						Title:   "Task Due Tomorrow",
						Message: "A task is due tomorrow!",
					},
				},
			},
		},
	}

	// Save the default config
	if err := l.Save(config); err != nil {
		return nil, err
	}

	// Create boards directory
	if err := os.MkdirAll(boardsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create boards directory: %w", err)
	}

	return config, nil
}

// GetConfigPath returns the path to the config file
func (l *Loader) GetConfigPath() string {
	return l.configPath
}
