package dto

// ColumnDTO represents a column data transfer object
type ColumnDTO struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Order       int       `json:"order"`
	WIPLimit    int       `json:"wip_limit"`
	Color       *string   `json:"color,omitempty"`
	Tasks       []TaskDTO `json:"tasks"`
	TaskCount   int       `json:"task_count"`
}

// CreateColumnRequest represents a request to create a column
type CreateColumnRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Order       int     `json:"order"`
	WIPLimit    int     `json:"wip_limit"`
	Color       *string `json:"color,omitempty"`
}

// UpdateColumnRequest represents a request to update a column
type UpdateColumnRequest struct {
	Description *string `json:"description,omitempty"`
	Order       *int    `json:"order,omitempty"`
	WIPLimit    *int    `json:"wip_limit,omitempty"`
	Color       *string `json:"color,omitempty"`
}
