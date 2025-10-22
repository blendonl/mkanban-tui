package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/serialization"
)

// ColumnStorage represents column storage format
type ColumnStorage struct {
	Description string `yaml:"description"`
	Order       int    `yaml:"order"`
	WIPLimit    int    `yaml:"wip_limit"`
	Color       string `yaml:"color,omitempty"`
}

// ColumnToStorage converts a Column entity to storage format
func ColumnToStorage(column *entity.Column) (map[string]interface{}, error) {
	frontmatter := map[string]interface{}{
		"description": column.Description(),
		"order":       column.Order(),
		"wip_limit":   column.WIPLimit(),
	}

	if column.Color() != nil {
		frontmatter["color"] = column.Color().String()
	}

	return frontmatter, nil
}

// ColumnFromStorage converts storage format to Column entity
func ColumnFromStorage(doc *serialization.FrontmatterDocument, name string) (*entity.Column, error) {
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

	column, err := entity.NewColumn(name, description, order, wipLimit, color)
	if err != nil {
		return nil, fmt.Errorf("failed to create column: %w", err)
	}

	return column, nil
}
