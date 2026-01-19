package valueobject

import (
	"fmt"
	"strings"
)

const boardIDSeparator = "/"

// BuildBoardID creates a composite board ID using the project slug and board slug.
func BuildBoardID(projectSlug, boardSlug string) (string, error) {
	projectSlug = strings.TrimSpace(projectSlug)
	boardSlug = strings.TrimSpace(boardSlug)
	if projectSlug == "" || boardSlug == "" {
		return "", fmt.Errorf("project slug and board slug are required")
	}
	return projectSlug + boardIDSeparator + boardSlug, nil
}

// ParseBoardID splits a composite board ID into project slug and board slug.
func ParseBoardID(boardID string) (string, string, error) {
	parts := strings.SplitN(boardID, boardIDSeparator, 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid board ID: %s", boardID)
	}
	return parts[0], parts[1], nil
}
