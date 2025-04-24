package delpoy

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/om/delploy"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Git gitService

type gitService struct {
}

func init() {
	Git = gitService{}
}

const (
	GitRepo = "git_repo"
)

// CheckGit checks if the git command is available
// and returns an error if it is not.
func (c gitService) CheckGit(ctx *gin.Context) (bool, *errors.Error) {
	logger.Info(ctx, "Checking if git is available...")
	if result, err := delploy.RunCommand(ctx, "git", "--version"); err != nil {
		return false, errors.Verify("Git command not found", err)
	} else {
		logger.Info(ctx, fmt.Sprintf("Git version: %s", result))
	}

	return true, nil
}

// SetRepo sets the git repository URL
func (c gitService) SetRepo(ctx *gin.Context, repo string) *errors.Error {
	logger.Info(ctx, fmt.Sprintf("Setting git repository to: %s", repo))
	if !strings.HasPrefix(repo, "git@") {
		return errors.Verify("Only SSH addresses are supported")
	}

	cacheIns := cache.New[string](cache.JSON)
	if err := cacheIns.Set(GitRepo, repo, 0); err != nil {
		return errors.Verify("Failed to set git repository")
	}
	repo, err := cacheIns.Get(GitRepo)
	if err != nil {
		return errors.Verify("Failed to get git repository")
	}
	if repo != "" {
		logger.Info(ctx, fmt.Sprintf("Git repository set to: %s", repo))
	} else {
		return errors.Verify("Git repository is empty")
	}
	return nil
}

// GetSSHKey retrieves the SSH key for the git repository
func (c gitService) GetSSHKey(ctx *gin.Context) (string, *errors.Error) {
	logger.Info(ctx, "Retrieving SSH key for git repository...")
	// 先查询本地是否存在 SSH key 一般在 ~/.ssh/id_rsa.pub
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Verify("Failed to get user home directory", err)
	}

	sshPath := filepath.Join(homeDir, ".ssh", "id_rsa.pub")

	sshKey := ""
	if _, errExist := os.Stat(sshPath); !os.IsNotExist(errExist) {
		if result, errR := delploy.RunCommand(ctx, "cat", sshPath); errR != nil {
			return "", errors.Verify("Failed to read SSH key", err)
		} else {
			sshKey = result
		}
	}

	if sshKey == "" {
		hostname, errH := delploy.RunCommand(ctx, "hostname")
		if errH != nil {
			return "", errors.Verify("Failed to retrieve hostname", errH)
		}
		if hostname == "" {
			hostname = fmt.Sprintf("gen_%d", time.Now().Unix())
		}

		err = nil
		sshKey, err = delploy.RunCommand(ctx, "ssh-keygen", "-t", "rsa", "-b", "4096", "-C", fmt.Sprintf("%s@%s", "gorig", hostname))
		if err != nil {
			return "", errors.Verify("Failed to generate SSH key", err)
		}
	}

	logger.Info(ctx, fmt.Sprintf("SSH key retrieved: %s", sshKey))

	return sshKey, nil
}

// ListBranches lists all branches in the git repository
func (c gitService) ListBranches(ctx *gin.Context) ([]string, *errors.Error) {
	logger.Info(ctx, "Listing all branches in the git repository...")
	// 查询 repo 的远程分支
	repo := cache.New[string](cache.JSON)
	if repo == nil {
		return nil, errors.Verify("Failed to get git repository")
	}
	repoURL, err := repo.Get(GitRepo)
	if err != nil {
		return nil, errors.Verify("Failed to get git repository")
	}
	if repoURL == "" {
		return nil, errors.Verify("Git repository is empty")
	}

	// git", "ls-remote", "--heads", repoURL
	branches, errR := delploy.RunCommand(ctx, "git", "ls-remote", "--heads", repoURL)
	if errR != nil {
		return nil, errors.Verify("Failed to list branches", errR)
	}

	branchList := strings.Split(branches, "\n")
	var branchNames []string

	for _, branch := range branchList {
		if strings.Contains(branch, "refs/heads/") {
			if strings.Contains(branch, "\t") {
				branch = strings.Split(branch, "\t")[1]
			}
			branchName := strings.TrimPrefix(branch, "refs/heads/")
			branchNames = append(branchNames, branchName)
		}
	}

	logger.Info(ctx, fmt.Sprintf("Branches found: %v", branchNames))
	return branchNames, nil
}

// AssociateBranch associates the current branch with the specified branch
func (c gitService) AssociateBranch(ctx *gin.Context, branch string) *errors.Error {
	logger.Info(ctx, fmt.Sprintf("Associating current branch with: %s", branch))
	if branch == "" {
		return errors.Verify("Branch name cannot be empty")
	}

	cacheIns := cache.New[string](cache.JSON)
	if err := cacheIns.Set("current_branch", branch, 0); err != nil {
		return errors.Verify("Failed to set current branch")
	}

	currentBranch, err := cacheIns.Get("current_branch")
	if err != nil {
		return errors.Verify("Failed to get current branch")
	}

	logger.Info(ctx, fmt.Sprintf("Current branch associated with: %s", currentBranch))
	return nil
}
