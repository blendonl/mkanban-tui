package dto

import (
	"mkanban/internal/domain/entity"
)

// BoardToDTO converts a Board entity to BoardDTO
func BoardToDTO(board *entity.Board) BoardDTO {
	columns := make([]ColumnDTO, 0, len(board.Columns()))
	for _, col := range board.Columns() {
		columns = append(columns, ColumnToDTO(col))
	}

	return BoardDTO{
		ID:          board.ID(),
		Name:        board.Name(),
		Prefix:      board.Prefix(),
		Description: board.Description(),
		Columns:     columns,
		CreatedAt:   board.CreatedAt(),
		ModifiedAt:  board.ModifiedAt(),
	}
}

// BoardToListDTO converts a Board entity to BoardListDTO
func BoardToListDTO(board *entity.Board) BoardListDTO {
	return BoardListDTO{
		ID:          board.ID(),
		Name:        board.Name(),
		Description: board.Description(),
		TaskCount:   board.TotalTaskCount(),
		ColumnCount: board.ColumnCount(),
		ModifiedAt:  board.ModifiedAt(),
	}
}

// ColumnToDTO converts a Column entity to ColumnDTO
func ColumnToDTO(column *entity.Column) ColumnDTO {
	tasks := make([]TaskDTO, 0, len(column.Tasks()))
	for _, task := range column.Tasks() {
		tasks = append(tasks, TaskToDTO(task))
	}

	var color *string
	if column.Color() != nil {
		colorStr := column.Color().String()
		color = &colorStr
	}

	return ColumnDTO{
		Name:        column.DisplayName(), // Use display name for UI
		Description: column.Description(),
		Order:       column.Order(),
		WIPLimit:    column.WIPLimit(),
		Color:       color,
		Tasks:       tasks,
		TaskCount:   column.TaskCount(),
	}
}

// TaskToDTO converts a Task entity to TaskDTO
func TaskToDTO(task *entity.Task) TaskDTO {
	return TaskDTO{
		ID:            task.ID().String(),
		ShortID:       task.ID().ShortID(),
		Title:         task.Title(),
		Description:   task.Description(),
		Priority:      task.Priority().String(),
		Status:        task.Status().String(),
		Tags:          task.Tags(),
		CreatedAt:     task.CreatedAt(),
		ModifiedAt:    task.ModifiedAt(),
		DueDate:       task.DueDate(),
		CompletedDate: task.CompletedDate(),
		IsOverdue:     task.IsOverdue(),
	}
}

// TaskToDTOWithPath converts a Task entity to TaskDTO with file path and column name
func TaskToDTOWithPath(task *entity.Task, filePath string, columnName string) TaskDTO {
	dto := TaskToDTO(task)
	dto.FilePath = filePath
	dto.ColumnName = columnName
	return dto
}
