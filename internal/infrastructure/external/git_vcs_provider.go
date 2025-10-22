package external

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitVCSProvider implements VCSProvider for Git
type GitVCSProvider struct{}

// NewGitVCSProvider creates a new GitVCSProvider
func NewGitVCSProvider() *GitVCSProvider {
	return &GitVCSProvider{}
}

// IsRepository checks if the given path is inside a git repository
func (g *GitVCSProvider) IsRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run() == nil
}

// GetRepositoryRoot returns the root directory of the git repository
func (g *GitVCSProvider) GetRepositoryRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	root := strings.TrimSpace(out.String())
	if root == "" {
		return "", fmt.Errorf("empty repository root for path: %s", path)
	}

	return root, nil
}

// GetCurrentBranch returns the name of the currently checked out branch
func (g *GitVCSProvider) GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(out.String())
	if branch == "" {
		return "", fmt.Errorf("empty branch name")
	}

	// Handle detached HEAD state
	if branch == "HEAD" {
		// Get the commit hash instead
		cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
		cmd.Dir = repoPath
		out.Reset()
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err != nil {
			return "HEAD", nil
		}

		commit := strings.TrimSpace(out.String())
		if commit != "" {
			return fmt.Sprintf("HEAD (detached at %s)", commit), nil
		}
	}

	return branch, nil
}

// ListBranches returns a list of all local branches in the repository
func (g *GitVCSProvider) ListBranches(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	branches := make([]string, 0, len(lines))

	for _, line := range lines {
		branch := strings.TrimSpace(line)
		if branch != "" {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

// GetRefsPath returns the path to the git refs directory for watching
func (g *GitVCSProvider) GetRefsPath(repoPath string) string {
	// Get the .git directory
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = repoPath

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return ""
	}

	gitDir := strings.TrimSpace(out.String())
	if gitDir == "" {
		return ""
	}

	// Handle relative .git directory
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoPath, gitDir)
	}

	// Return the refs/heads path for watching local branch changes
	refsPath := filepath.Join(gitDir, "refs", "heads")

	// Verify the path exists
	if _, err := os.Stat(refsPath); err != nil {
		return ""
	}

	return refsPath
}
