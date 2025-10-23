package style

import (
	"github.com/charmbracelet/lipgloss"
	"mkanban/internal/infrastructure/config"
)

var (
	ColumnStyle       lipgloss.Style
	FocusedColumnStyle lipgloss.Style
	ColumnTitleStyle  lipgloss.Style
	TaskStyle         lipgloss.Style
	SelectedTaskStyle lipgloss.Style
	HelpStyle         lipgloss.Style

	// New task card component styles
	DescriptionStyle      lipgloss.Style
	TagStyle             lipgloss.Style
	DueDateStyle         lipgloss.Style
	OverdueStyle         lipgloss.Style
	TaskCardStyle        lipgloss.Style
	SelectedTaskCardStyle lipgloss.Style
	ScrollIndicatorStyle  lipgloss.Style
)

// InitStyles initializes the styles from config
func InitStyles(cfg *config.Config) {
	styles := cfg.TUI.Styles

	// Column style
	ColumnStyle = lipgloss.NewStyle().
		Padding(styles.Column.PaddingVertical, styles.Column.PaddingHorizontal).
		Border(getBorder(styles.Column.BorderStyle)).
		BorderForeground(lipgloss.Color(styles.Column.BorderColor))

	// Focused column style
	FocusedColumnStyle = lipgloss.NewStyle().
		Padding(styles.FocusedColumn.PaddingVertical, styles.FocusedColumn.PaddingHorizontal).
		Border(getBorder(styles.FocusedColumn.BorderStyle)).
		BorderForeground(lipgloss.Color(styles.FocusedColumn.BorderColor))

	// Column title style
	ColumnTitleStyle = lipgloss.NewStyle()
	if styles.ColumnTitle.Foreground != "" {
		ColumnTitleStyle = ColumnTitleStyle.Foreground(lipgloss.Color(styles.ColumnTitle.Foreground))
	}
	if styles.ColumnTitle.Bold {
		ColumnTitleStyle = ColumnTitleStyle.Bold(true)
	}
	if styles.ColumnTitle.Align != "" {
		ColumnTitleStyle = ColumnTitleStyle.Align(getAlign(styles.ColumnTitle.Align))
	}

	// Task style
	TaskStyle = lipgloss.NewStyle().
		Padding(styles.Task.PaddingVertical, styles.Task.PaddingHorizontal)
	if styles.Task.Foreground != "" {
		TaskStyle = TaskStyle.Foreground(lipgloss.Color(styles.Task.Foreground))
	}

	// Selected task style
	SelectedTaskStyle = lipgloss.NewStyle().
		Padding(styles.SelectedTask.PaddingVertical, styles.SelectedTask.PaddingHorizontal)
	if styles.SelectedTask.Foreground != "" {
		SelectedTaskStyle = SelectedTaskStyle.Foreground(lipgloss.Color(styles.SelectedTask.Foreground))
	}
	if styles.SelectedTask.Background != "" {
		SelectedTaskStyle = SelectedTaskStyle.Background(lipgloss.Color(styles.SelectedTask.Background))
	}
	if styles.SelectedTask.Bold {
		SelectedTaskStyle = SelectedTaskStyle.Bold(true)
	}

	// Help style
	HelpStyle = lipgloss.NewStyle().
		Padding(styles.Help.PaddingVertical, 0, 0, styles.Help.PaddingHorizontal)
	if styles.Help.Foreground != "" {
		HelpStyle = HelpStyle.Foreground(lipgloss.Color(styles.Help.Foreground))
	}

	// Task card component styles
	DescriptionStyle = lipgloss.NewStyle().
		Padding(0, 0, 0, styles.Description.PaddingHorizontal)
	if styles.Description.Foreground != "" {
		DescriptionStyle = DescriptionStyle.Foreground(lipgloss.Color(styles.Description.Foreground))
	}
	if styles.Description.Italic {
		DescriptionStyle = DescriptionStyle.Italic(true)
	}

	TagStyle = lipgloss.NewStyle().
		Padding(0, 0, 0, styles.Tag.PaddingHorizontal)
	if styles.Tag.Foreground != "" {
		TagStyle = TagStyle.Foreground(lipgloss.Color(styles.Tag.Foreground))
	}

	DueDateStyle = lipgloss.NewStyle().
		Padding(0, 0, 0, styles.DueDate.PaddingHorizontal)
	if styles.DueDate.Foreground != "" {
		DueDateStyle = DueDateStyle.Foreground(lipgloss.Color(styles.DueDate.Foreground))
	}

	OverdueStyle = lipgloss.NewStyle().
		Padding(0, 0, 0, styles.Overdue.PaddingHorizontal)
	if styles.Overdue.Foreground != "" {
		OverdueStyle = OverdueStyle.Foreground(lipgloss.Color(styles.Overdue.Foreground))
	}
	if styles.Overdue.Bold {
		OverdueStyle = OverdueStyle.Bold(true)
	}

	// Task card container styles
	TaskCardStyle = lipgloss.NewStyle().
		Padding(0, 1).
		MarginBottom(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(styles.TaskCard.BorderColor))

	SelectedTaskCardStyle = lipgloss.NewStyle().
		Padding(0, 1).
		MarginBottom(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(styles.SelectedTaskCard.BorderColor))

	// Scroll indicator style
	ScrollIndicatorStyle = lipgloss.NewStyle().
		Italic(true).
		Align(lipgloss.Center)
	if styles.ScrollIndicator.Foreground != "" {
		ScrollIndicatorStyle = ScrollIndicatorStyle.Foreground(lipgloss.Color(styles.ScrollIndicator.Foreground))
	}
	if styles.ScrollIndicator.Bold {
		ScrollIndicatorStyle = ScrollIndicatorStyle.Bold(true)
	}
}

// getBorder returns the border style based on the name
func getBorder(name string) lipgloss.Border {
	switch name {
	case "rounded":
		return lipgloss.RoundedBorder()
	case "normal":
		return lipgloss.NormalBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	default:
		return lipgloss.RoundedBorder()
	}
}

// getAlign returns the alignment based on the name
func getAlign(name string) lipgloss.Position {
	switch name {
	case "left":
		return lipgloss.Left
	case "center":
		return lipgloss.Center
	case "right":
		return lipgloss.Right
	default:
		return lipgloss.Center
	}
}
