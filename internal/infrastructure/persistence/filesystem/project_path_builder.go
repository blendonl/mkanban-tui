package filesystem

import "path/filepath"

const (
	projectsDir       = "projects"
	projectMetadataFile = "project.md"
	boardsDir         = "boards"
	notesDir          = "notes"
	timeDir           = "time"
	timeLogsDir       = "logs"
)

type ProjectPathBuilder struct {
	rootPath string
}

func NewProjectPathBuilder(rootPath string) *ProjectPathBuilder {
	return &ProjectPathBuilder{rootPath: rootPath}
}

func (pb *ProjectPathBuilder) ProjectsRoot() string {
	return filepath.Join(pb.rootPath, projectsDir)
}

func (pb *ProjectPathBuilder) ProjectDir(projectSlug string) string {
	return filepath.Join(pb.ProjectsRoot(), projectSlug)
}

func (pb *ProjectPathBuilder) ProjectMetadata(projectSlug string) string {
	return filepath.Join(pb.ProjectDir(projectSlug), projectMetadataFile)
}

func (pb *ProjectPathBuilder) ProjectBoardsDir(projectSlug string) string {
	return filepath.Join(pb.ProjectDir(projectSlug), boardsDir)
}

func (pb *ProjectPathBuilder) ProjectNotesDir(projectSlug string) string {
	return filepath.Join(pb.ProjectDir(projectSlug), notesDir)
}

func (pb *ProjectPathBuilder) ProjectTimeDir(projectSlug string) string {
	return filepath.Join(pb.ProjectDir(projectSlug), timeDir)
}

func (pb *ProjectPathBuilder) ProjectTimeLogsDir(projectSlug string) string {
	return filepath.Join(pb.ProjectTimeDir(projectSlug), timeLogsDir)
}

func (pb *ProjectPathBuilder) TimeLogFile(projectSlug string, yearMonth string) string {
	return filepath.Join(pb.ProjectTimeLogsDir(projectSlug), yearMonth+".yml")
}

func (pb *ProjectPathBuilder) GlobalDir() string {
	return filepath.Join(pb.rootPath, "global")
}

func (pb *ProjectPathBuilder) GlobalNotesDir() string {
	return filepath.Join(pb.GlobalDir(), notesDir)
}

func (pb *ProjectPathBuilder) GlobalTimeDir() string {
	return filepath.Join(pb.GlobalDir(), timeDir)
}
