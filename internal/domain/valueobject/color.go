package valueobject

import (
	"fmt"
	"regexp"
	"strings"
)

// Color represents a hex color value
type Color struct {
	value string // Hex color in format #RRGGBB or #RGB
}

var hexColorRegex = regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)

// NewColor creates a new Color value object
func NewColor(hex string) (*Color, error) {
	if hex == "" {
		return nil, fmt.Errorf("color cannot be empty")
	}

	if !hexColorRegex.MatchString(hex) {
		return nil, fmt.Errorf("invalid color format: must be #RGB or #RRGGBB")
	}

	return &Color{value: strings.ToUpper(hex)}, nil
}

// String returns the hex color string
func (c *Color) String() string {
	return c.value
}

// Equal checks if two colors are equal
func (c *Color) Equal(other *Color) bool {
	if other == nil {
		return false
	}
	return c.value == other.value
}

// MarshalText implements encoding.TextMarshaler
func (c *Color) MarshalText() ([]byte, error) {
	return []byte(c.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (c *Color) UnmarshalText(text []byte) error {
	color, err := NewColor(string(text))
	if err != nil {
		return err
	}
	c.value = color.value
	return nil
}

// Predefined colors
var (
	ColorBlue   = &Color{value: "#3498DB"}
	ColorGreen  = &Color{value: "#2ECC71"}
	ColorYellow = &Color{value: "#F1C40F"}
	ColorRed    = &Color{value: "#E74C3C"}
	ColorPurple = &Color{value: "#9B59B6"}
	ColorGray   = &Color{value: "#95A5A6"}
)
