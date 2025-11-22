package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Printer provides methods for formatted console output
type Printer struct {
	writer io.Writer
	styles *Styles
}

// Styles holds lipgloss styles for console output
type Styles struct {
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Header  lipgloss.Style
	Subtle  lipgloss.Style
	Bold    lipgloss.Style
}

// NewPrinter creates a new console printer
func NewPrinter(writer io.Writer) *Printer {
	return &Printer{
		writer: writer,
		styles: &Styles{
			Success: lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true),
			Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
			Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true),
			Info:    lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true),
			Header:  lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Underline(true),
			Subtle:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			Bold:    lipgloss.NewStyle().Bold(true),
		},
	}
}

// Success prints a success message
func (p *Printer) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, p.styles.Success.Render("✓ "+msg))
}

// Error prints an error message
func (p *Printer) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, p.styles.Error.Render("✗ "+msg))
}

// Warning prints a warning message
func (p *Printer) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, p.styles.Warning.Render("⚠ "+msg))
}

// Info prints an info message
func (p *Printer) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, p.styles.Info.Render("ℹ "+msg))
}

// Header prints a header message
func (p *Printer) Header(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, p.styles.Header.Render(msg))
}

// Println prints a normal message
func (p *Printer) Println(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, msg)
}

// Print prints a normal message without newline
func (p *Printer) Print(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprint(p.writer, msg)
}

// Subtle prints a subtle/dimmed message
func (p *Printer) Subtle(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, p.styles.Subtle.Render(msg))
}

// Bold prints a bold message
func (p *Printer) Bold(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.writer, p.styles.Bold.Render(msg))
}

// Table prints a simple table
func (p *Printer) Table(headers []string, rows [][]string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	headerParts := make([]string, len(headers))
	for i, h := range headers {
		headerParts[i] = p.styles.Bold.Render(padRight(h, widths[i]))
	}
	fmt.Fprintln(p.writer, strings.Join(headerParts, "  "))

	// Print separator
	separatorParts := make([]string, len(headers))
	for i, w := range widths {
		separatorParts[i] = strings.Repeat("-", w)
	}
	fmt.Fprintln(p.writer, p.styles.Subtle.Render(strings.Join(separatorParts, "  ")))

	// Print rows
	for _, row := range rows {
		rowParts := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				rowParts[i] = padRight(row[i], widths[i])
			} else {
				rowParts[i] = padRight("", widths[i])
			}
		}
		fmt.Fprintln(p.writer, strings.Join(rowParts, "  "))
	}
}

// padRight pads a string to the right
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// DefaultPrinter returns a printer that writes to stdout
func DefaultPrinter() *Printer {
	return NewPrinter(os.Stdout)
}
