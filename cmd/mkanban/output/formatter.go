package output

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Format represents the output format type
type Format string

const (
	// FormatText is the default text output format
	FormatText Format = "text"
	// FormatJSON is JSON output format
	FormatJSON Format = "json"
	// FormatYAML is YAML output format
	FormatYAML Format = "yaml"
)

// Formatter handles output formatting for different formats
type Formatter struct {
	format Format
	writer io.Writer
}

// NewFormatter creates a new output formatter
func NewFormatter(format Format, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

// Print outputs data in the configured format
func (f *Formatter) Print(data interface{}) error {
	switch f.format {
	case FormatJSON:
		return f.printJSON(data)
	case FormatYAML:
		return f.printYAML(data)
	case FormatText:
		// For text format, expect the data to be a string or implement fmt.Stringer
		return f.printText(data)
	default:
		return fmt.Errorf("unsupported output format: %s", f.format)
	}
}

// printJSON outputs data as JSON
func (f *Formatter) printJSON(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML outputs data as YAML
func (f *Formatter) printYAML(data interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// printText outputs data as plain text
func (f *Formatter) printText(data interface{}) error {
	switch v := data.(type) {
	case string:
		_, err := fmt.Fprintln(f.writer, v)
		return err
	case fmt.Stringer:
		_, err := fmt.Fprintln(f.writer, v.String())
		return err
	default:
		_, err := fmt.Fprintln(f.writer, v)
		return err
	}
}

// ParseFormat converts a string to a Format
func ParseFormat(s string) (Format, error) {
	switch s {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return FormatText, fmt.Errorf("invalid format '%s': must be one of: text, json, yaml", s)
	}
}
