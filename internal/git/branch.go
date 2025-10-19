package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetAllBranches returns a list of all branches (local and remote)
func GetAllBranches(path string) ([]string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	// Get all branches (local and remote)
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var branches []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove the * prefix for current branch and remote prefixes
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "remotes/")

		// Skip HEAD references
		if strings.Contains(line, "HEAD") {
			continue
		}

		// Remove origin/ prefix for cleaner display
		line = strings.TrimPrefix(line, "origin/")

		// Avoid duplicates
		exists := false
		for _, existing := range branches {
			if existing == line {
				exists = true
				break
			}
		}
		if !exists {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// GetRemoteBranches returns a list of remote branches
func GetRemoteBranches(path string) ([]string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	// Get remote branches
	cmd := exec.Command("git", "branch", "-r")
	cmd.Dir = workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get remote branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var branches []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip HEAD references
		if strings.Contains(line, "HEAD") {
			continue
		}

		// Remove remotes/ prefix for cleaner display
		line = strings.TrimPrefix(line, "remotes/")

		// Avoid duplicates
		exists := false
		for _, existing := range branches {
			if existing == line {
				exists = true
				break
			}
		}
		if !exists {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// GetBranchDiff returns the diff between two branches
func GetBranchDiff(path, fromBranch, toBranch string) (string, error) {
	// Validate inputs
	if fromBranch == "" {
		return "", fmt.Errorf("source branch name is empty")
	}
	if toBranch == "" {
		return "", fmt.Errorf("target branch name is empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	// Use a safer approach - compare the branches directly
	cmd := exec.Command("git", "diff", "-U5", "--no-color", fromBranch, toBranch)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 && stdout.Len() > 0 {
				return stdout.String(), nil
			}
		}

		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("git diff failed: %s", errMsg)
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	return stdout.String(), nil
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
