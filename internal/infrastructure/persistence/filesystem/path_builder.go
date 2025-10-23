package filesystem

import (
	"path/filepath"
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
	boardsRootPath string
}

// NewPathBuilder creates a new PathBuilder
func NewPathBuilder(boardsRootPath string) *PathBuilder {
	return &PathBuilder{
		boardsRootPath: boardsRootPath,
	}
}

// BoardsRoot returns the root path for all boards
func (pb *PathBuilder) BoardsRoot() string {
	return pb.boardsRootPath
}

// BoardDir returns the directory path for a board
func (pb *PathBuilder) BoardDir(boardID string) string {
	return filepath.Join(pb.boardsRootPath, boardID)
}

// BoardMetadataYaml returns the path to a board's metadata.yml file
func (pb *PathBuilder) BoardMetadataYaml(boardID string) string {
	return filepath.Join(pb.BoardDir(boardID), boardMetadataYamlFile)
}

// BoardContent returns the path to a board's board.md file
func (pb *PathBuilder) BoardContent(boardID string) string {
	return filepath.Join(pb.BoardDir(boardID), boardContentFile)
}

// ColumnDir returns the directory path for a column
func (pb *PathBuilder) ColumnDir(boardID string, columnName string) string {
	return filepath.Join(pb.BoardDir(boardID), "columns", columnName)
}

// ColumnMetadataYaml returns the path to a column's metadata.yml file
func (pb *PathBuilder) ColumnMetadataYaml(boardID string, columnName string) string {
	return filepath.Join(pb.ColumnDir(boardID, columnName), columnMetadataYamlFile)
}

// ColumnContent returns the path to a column's column.md file
func (pb *PathBuilder) ColumnContent(boardID string, columnName string) string {
	return filepath.Join(pb.ColumnDir(boardID, columnName), columnContentFile)
}

// TaskDir returns the directory path for a task
func (pb *PathBuilder) TaskDir(boardID string, columnName string, taskFolderName string) string {
	return filepath.Join(pb.ColumnDir(boardID, columnName), "tasks", taskFolderName)
}

// TaskMetadata returns the path to a task's metadata file
func (pb *PathBuilder) TaskMetadata(boardID string, columnName string, taskFolderName string) string {
	return filepath.Join(pb.TaskDir(boardID, columnName, taskFolderName), taskMetadataFile)
}

// TaskMetadataYaml returns the path to a task's metadata.yml file
func (pb *PathBuilder) TaskMetadataYaml(boardID string, columnName string, taskFolderName string) string {
	return filepath.Join(pb.TaskDir(boardID, columnName, taskFolderName), taskMetadataYamlFile)
}
