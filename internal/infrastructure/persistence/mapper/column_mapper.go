package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/serialization"
)

// ColumnStorage represents column metadata storage format (metadata.yml)
type ColumnStorage struct {
	Order    int    `yaml:"order"`
	WIPLimit int    `yaml:"wip_limit"`
	Color    string `yaml:"color,omitempty"`
}

// ColumnMetadataToStorage converts a Column entity to metadata storage format
func ColumnMetadataToStorage(column *entity.Column) (map[string]interface{}, error) {
	metadata := map[string]interface{}{
		"order":     column.Order(),
		"wip_limit": column.WIPLimit(),
	}

	if column.Color() != nil {
		metadata["color"] = column.Color().String()
	}

	return metadata, nil
}

// ColumnContentToMarkdown converts a Column entity to markdown content (column.md)
func ColumnContentToMarkdown(column *entity.Column) []byte {
	return serialization.SerializeMarkdownWithTitle(column.DisplayName(), column.Description())
}

// ColumnFromStorage converts storage format to Column entity (new split format)
func ColumnFromStorage(metadataDoc *serialization.FrontmatterDocument, name string, displayName string, description string) (*entity.Column, error) {
	order := metadataDoc.GetInt("order")
	wipLimit := metadataDoc.GetInt("wip_limit")

	var color *valueobject.Color
	colorStr := metadataDoc.GetString("color")
	if colorStr != "" {
		var err error
		color, err = valueobject.NewColor(colorStr)
		if err != nil {
			// If color is invalid, just use nil
			color = nil
		}
	}

	column, err := entity.NewColumnWithDisplayName(name, displayName, description, order, wipLimit, color)
	if err != nil {
		return nil, fmt.Errorf("failed to create column: %w", err)
	}

	return column, nil
}

// ColumnFromLegacyStorage converts old frontmatter format to Column entity (backward compatibility)
func ColumnFromLegacyStorage(doc *serialization.FrontmatterDocument, name string) (*entity.Column, error) {
	displayName := doc.GetString("display_name")
	if displayName == "" {
		// Fallback for old format: use folder name as display name
		displayName = name
	}
	description := doc.GetString("description")
	order := doc.GetInt("order")
	wipLimit := doc.GetInt("wip_limit")

	var color *valueobject.Color
	colorStr := doc.GetString("color")
	if colorStr != "" {
		var err error
		color, err = valueobject.NewColor(colorStr)
		if err != nil {
			// If color is invalid, just use nil
			color = nil
		}
	}

	column, err := entity.NewColumnWithDisplayName(name, displayName, description, order, wipLimit, color)
	if err != nil {
		return nil, fmt.Errorf("failed to create column: %w", err)
	}

	return column, nil
}
