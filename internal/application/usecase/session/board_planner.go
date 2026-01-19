package session

import (
	"path/filepath"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
	"mkanban/pkg/filesystem"
)

type SessionBoardPlan struct {
	ProjectID    string
	ProjectName  string
	WorkingDir   string
	BoardNames   []string
	SyncBoardName string
	IsRepo       bool
}

type SessionBoardPlanner struct {
	vcsProvider service.VCSProvider
}

func NewSessionBoardPlanner(vcsProvider service.VCSProvider) *SessionBoardPlanner {
	return &SessionBoardPlanner{
		vcsProvider: vcsProvider,
	}
}

func (p *SessionBoardPlanner) Plan(session *entity.Session) (*SessionBoardPlan, error) {
	workingDir := session.WorkingDir()
	projectName := session.Name()
	projectWorkingDir := workingDir

	isRepo := p.vcsProvider.IsRepository(workingDir)
	if isRepo {
		repoRoot, err := p.vcsProvider.GetRepositoryRoot(workingDir)
		if err != nil {
			return nil, err
		}
		projectWorkingDir = repoRoot
		projectName = filepath.Base(repoRoot)
	}

	projectID := valueobject.GenerateSlug(projectName)
	boardNames := []string{"default"}

	if isRepo {
		monorepo, err := p.isMonorepo(projectWorkingDir)
		if err != nil {
			return nil, err
		}
		if monorepo {
			boardNames = []string{"default", "backend", "android"}
		}
	}

	return &SessionBoardPlan{
		ProjectID:     projectID,
		ProjectName:   projectName,
		WorkingDir:    projectWorkingDir,
		BoardNames:    boardNames,
		SyncBoardName: "default",
		IsRepo:        isRepo,
	}, nil
}

func (p *SessionBoardPlanner) isMonorepo(repoRoot string) (bool, error) {
	backendPath := filepath.Join(repoRoot, "backend")
	backendExists, err := filesystem.Exists(backendPath)
	if err != nil {
		return false, err
	}

	mobilePath := filepath.Join(repoRoot, "mobile")
	mobileExists, err := filesystem.Exists(mobilePath)
	if err != nil {
		return false, err
	}

	androidPath := filepath.Join(repoRoot, "android")
	androidExists, err := filesystem.Exists(androidPath)
	if err != nil {
		return false, err
	}

	return backendExists && (mobileExists || androidExists), nil
}
