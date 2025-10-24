package service

// VCSProvider defines the interface for interacting with version control systems
// This abstraction allows for different VCS implementations (git, svn, hg, etc.)
type VCSProvider interface {
	// IsRepository checks if the given path is a VCS repository
	IsRepository(path string) bool

	// GetRepositoryRoot returns the root directory of the repository
	// Returns empty string if path is not in a repository
	GetRepositoryRoot(path string) (string, error)

	// GetCurrentBranch returns the name of the currently checked out branch
	GetCurrentBranch(repoPath string) (string, error)

	// ListBranches returns a list of all local branches in the repository
	ListBranches(repoPath string) ([]string, error)

	// GetRefsPath returns the path to the refs directory for watching
	// (e.g., .git/refs/heads/ for git)
	GetRefsPath(repoPath string) string

	// BranchExists checks if a branch with the given name exists
	BranchExists(repoPath, branchName string) (bool, error)

	// CheckoutBranch checks out an existing branch
	CheckoutBranch(repoPath, branchName string) error

	// CreateAndCheckoutBranch creates a new branch and checks it out
	CreateAndCheckoutBranch(repoPath, branchName string) error
}
