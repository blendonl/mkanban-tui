package filesystem

import (
	"path/filepath"

	"mkanban/internal/domain/valueobject"
)

const (
	boardMetadataYamlFile  = "metadata.yml"
	boardContentFile       = "board.md"
	columnMetadataYamlFile = "metadata.yml"
	columnContentFile      = "column.md"
	taskMetadataFile       = "task.md"
	taskMetadataYamlFile   = "metadata.yml"
)

// PathBuilder constructs filesystem paths for board entities
type PathBuilder struct {
	projectPathBuilder *ProjectPathBuilder
}

// NewPathBuilder creates a new PathBuilder
func NewPathBuilder(rootPath string) *PathBuilder {
	return &PathBuilder{
		projectPathBuilder: NewProjectPathBuilder(rootPath),
	}
}

// BoardDir returns the directory path for a board
func (pb *PathBuilder) BoardDir(boardID string) (string, error) {
	projectSlug, boardSlug, err := valueobject.ParseBoardID(boardID)
	if err != nil {
		return "", err
	}
	return filepath.Join(pb.projectPathBuilder.ProjectBoardsDir(projectSlug), boardSlug), nil
}

// BoardMetadataYaml returns the path to a board's metadata.yml file
func (pb *PathBuilder) BoardMetadataYaml(boardID string) (string, error) {
	boardDir, err := pb.BoardDir(boardID)
	if err != nil {
		return "", err
	}
	return filepath.Join(boardDir, boardMetadataYamlFile), nil
}

// BoardContent returns the path to a board's board.md file
func (pb *PathBuilder) BoardContent(boardID string) (string, error) {
	boardDir, err := pb.BoardDir(boardID)
	if err != nil {
		return "", err
	}
	return filepath.Join(boardDir, boardContentFile), nil
}

// ColumnDir returns the directory path for a column
func (pb *PathBuilder) ColumnDir(boardID string, columnName string) (string, error) {
	boardDir, err := pb.BoardDir(boardID)
	if err != nil {
		return "", err
	}
	return filepath.Join(boardDir, "columns", columnName), nil
}

// ColumnMetadataYaml returns the path to a column's metadata.yml file
func (pb *PathBuilder) ColumnMetadataYaml(boardID string, columnName string) (string, error) {
	columnDir, err := pb.ColumnDir(boardID, columnName)
	if err != nil {
		return "", err
	}
	return filepath.Join(columnDir, columnMetadataYamlFile), nil
}

// ColumnContent returns the path to a column's column.md file
func (pb *PathBuilder) ColumnContent(boardID string, columnName string) (string, error) {
	columnDir, err := pb.ColumnDir(boardID, columnName)
	if err != nil {
		return "", err
	}
	return filepath.Join(columnDir, columnContentFile), nil
}

// TaskDir returns the directory path for a task
func (pb *PathBuilder) TaskDir(boardID string, columnName string, taskFolderName string) (string, error) {
	columnDir, err := pb.ColumnDir(boardID, columnName)
	if err != nil {
		return "", err
	}
	return filepath.Join(columnDir, "tasks", taskFolderName), nil
}

// TaskMetadata returns the path to a task's metadata file
func (pb *PathBuilder) TaskMetadata(boardID string, columnName string, taskFolderName string) (string, error) {
	taskDir, err := pb.TaskDir(boardID, columnName, taskFolderName)
	if err != nil {
		return "", err
	}
	return filepath.Join(taskDir, taskMetadataFile), nil
}

// TaskMetadataYaml returns the path to a task's metadata.yml file
func (pb *PathBuilder) TaskMetadataYaml(boardID string, columnName string, taskFolderName string) (string, error) {
	taskDir, err := pb.TaskDir(boardID, columnName, taskFolderName)
	if err != nil {
		return "", err
	}
	return filepath.Join(taskDir, taskMetadataYamlFile), nil
}
