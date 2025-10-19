package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StageAllFiles stages all files in the repository
func StageAllFiles(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = workDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all files: %w", err)
	}

	return nil
}

// CreateCommit creates a commit with the given message
func CreateCommit(path, message string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = workDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}

// HasStagedChanges checks if there are any staged changes
func HasStagedChanges(path string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	cmd := exec.Command("git", "diff", "--staged", "--quiet")
	cmd.Dir = workDir

	err = cmd.Run()
	// Exit code 1 means there are staged changes
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return true, nil
	}
	// Exit code 0 means no staged changes
	if err == nil {
		return false, nil
	}

	return false, fmt.Errorf("failed to check staged changes: %w", err)
}

// GetStagedFiles returns the list of staged files
func GetStagedFiles(path string) ([]string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}

	return files, nil
}

// PushBranch pushes the current branch to the remote repository
func PushBranch(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	cmd := exec.Command("git", "push")
	cmd.Dir = workDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	return nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
