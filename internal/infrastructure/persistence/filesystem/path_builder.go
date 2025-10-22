package filesystem

import (
	"path/filepath"
)

const (
	boardMetadataFile  = "board.md"
	columnMetadataFile = "column.md"
	taskMetadataFile   = "task.md"
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

// BoardMetadata returns the path to a board's metadata file
func (pb *PathBuilder) BoardMetadata(boardID string) string {
	return filepath.Join(pb.BoardDir(boardID), boardMetadataFile)
}

// ColumnDir returns the directory path for a column
func (pb *PathBuilder) ColumnDir(boardID string, columnName string) string {
	return filepath.Join(pb.BoardDir(boardID), columnName)
}

// ColumnMetadata returns the path to a column's metadata file
func (pb *PathBuilder) ColumnMetadata(boardID string, columnName string) string {
	return filepath.Join(pb.ColumnDir(boardID, columnName), columnMetadataFile)
}

// TaskDir returns the directory path for a task
func (pb *PathBuilder) TaskDir(boardID string, columnName string, taskFolderName string) string {
	return filepath.Join(pb.ColumnDir(boardID, columnName), taskFolderName)
}

// TaskMetadata returns the path to a task's metadata file
func (pb *PathBuilder) TaskMetadata(boardID string, columnName string, taskFolderName string) string {
	return filepath.Join(pb.TaskDir(boardID, columnName, taskFolderName), taskMetadataFile)
}
