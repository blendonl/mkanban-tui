package valueobject

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TaskID represents a unique task identifier in the format: PREFIX-NUMBER-slug
// Example: REC-001-setup-database
type TaskID struct {
	prefix string // Board prefix (3 letters uppercase)
	number int    // Sequential number
	slug   string // URL-safe slug from task title
}

var taskIDRegex = regexp.MustCompile(`^([A-Z]{3})-(\d+)-(.+)$`)

// NewTaskID creates a new TaskID from components
func NewTaskID(prefix string, number int, slug string) (*TaskID, error) {
	if len(prefix) != 3 {
		return nil, fmt.Errorf("prefix must be exactly 3 characters")
	}
	if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(prefix) {
		return nil, fmt.Errorf("prefix must be 3 uppercase letters")
	}
	if number < 1 {
		return nil, fmt.Errorf("number must be positive")
	}
	if slug == "" {
		return nil, fmt.Errorf("slug cannot be empty")
	}
	if !regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`).MatchString(slug) {
		return nil, fmt.Errorf("slug must be lowercase alphanumeric with hyphens")
	}

	return &TaskID{
		prefix: prefix,
		number: number,
		slug:   slug,
	}, nil
}

// ParseTaskID parses a task ID string
func ParseTaskID(id string) (*TaskID, error) {
	matches := taskIDRegex.FindStringSubmatch(id)
	if matches == nil {
		return nil, fmt.Errorf("invalid task ID format: %s", id)
	}

	number, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid task number: %w", err)
	}

	return &TaskID{
		prefix: matches[1],
		number: number,
		slug:   matches[3],
	}, nil
}

// String returns the full task ID string
func (t *TaskID) String() string {
	return fmt.Sprintf("%s-%03d-%s", t.prefix, t.number, t.slug)
}

// Prefix returns the board prefix
func (t *TaskID) Prefix() string {
	return t.prefix
}

// Number returns the task number
func (t *TaskID) Number() int {
	return t.number
}

// Slug returns the task slug
func (t *TaskID) Slug() string {
	return t.slug
}

// ShortID returns just the prefix and number (e.g., REC-001)
func (t *TaskID) ShortID() string {
	return fmt.Sprintf("%s-%03d", t.prefix, t.number)
}

// Equal checks if two TaskIDs are equal
func (t *TaskID) Equal(other *TaskID) bool {
	if other == nil {
		return false
	}
	return t.prefix == other.prefix &&
	       t.number == other.number &&
	       t.slug == other.slug
}

// GenerateBoardPrefix creates a 3-letter prefix from a board name
func GenerateBoardPrefix(boardName string) string {
	// Remove non-alphanumeric characters and convert to uppercase
	cleaned := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(boardName, "")
	upper := strings.ToUpper(cleaned)

	if len(upper) == 0 {
		return "BRD" // Default prefix
	}

	if len(upper) >= 3 {
		return upper[:3]
	}

	// Pad with 'X' if less than 3 characters
	return upper + strings.Repeat("X", 3-len(upper))
}
