package slug

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	multipleHyphensRegex = regexp.MustCompile(`-+`)
)

// Generate creates a URL-safe slug from a string
func Generate(s string) string {
	// Convert to lowercase
	slug := strings.ToLower(s)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove all non-alphanumeric characters except hyphens
	slug = nonAlphanumericRegex.ReplaceAllString(slug, "-")

	// Replace multiple consecutive hyphens with a single hyphen
	slug = multipleHyphensRegex.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// If empty after cleaning, return a default
	if slug == "" {
		return "untitled"
	}

	// Limit length to 50 characters
	if len(slug) > 50 {
		slug = slug[:50]
		// Trim any trailing hyphen after truncation
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}
