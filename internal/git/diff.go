package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsGitRepository checks if the given path is within a git repository
func IsGitRepository(path string) bool {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return false
	}

	// Check if we're in a git repository by running git rev-parse
	cmd := exec.Command("git", "rev-parse", "--git-dir")

	// If path is a file, use its directory
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	cmd.Dir = absPath
	err = cmd.Run()
	return err == nil
}

// GetDiff retrieves the git diff for the specified path
func GetDiff(path string, staged bool) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Set working directory
	workDir := absPath
	stat, err := os.Stat(absPath)
	if err == nil && !stat.IsDir() {
		workDir = filepath.Dir(absPath)
	}

	var allDiffs strings.Builder

	// Get regular diff
	regularDiff, err := getRegularDiff(absPath, workDir, staged)
	if err != nil {
		return "", err
	}
	allDiffs.WriteString(regularDiff)

	// If not showing only staged changes, also get new/untracked files
	if !staged {
		untrackedDiff, err := getUntrackedFilesDiff(workDir, absPath)
		if err == nil && untrackedDiff != "" {
			if allDiffs.Len() > 0 {
				allDiffs.WriteString("\n")
			}
			allDiffs.WriteString(untrackedDiff)
		}
	}

	return allDiffs.String(), nil
}

func getRegularDiff(absPath, workDir string, staged bool) (string, error) {
	// Prepare git diff command
	args := []string{"diff"}

	if staged {
		args = append(args, "--staged")
	} else {
		// Show all changes including staged ones
		args = append(args, "HEAD")
	}

	// Add unified context lines for better display
	args = append(args, "-U5")

	// No color - we'll apply our own
	args = append(args, "--no-color")

	// Add the path if it's not current directory
	if absPath != "." {
		args = append(args, "--", absPath)
	}

	// Execute git diff command
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		// git diff returns exit code 1 if there are differences, which is not an error
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check if it's just because there are diffs (exit code 1)
			if exitErr.ExitCode() == 1 && stdout.Len() > 0 {
				return stdout.String(), nil
			}
		}

		// Real error
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("git diff failed: %s", errMsg)
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	return stdout.String(), nil
}

func getUntrackedFilesDiff(workDir, filterPath string) (string, error) {
	// Get list of untracked files
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = workDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	untrackedFiles := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(untrackedFiles) == 0 || untrackedFiles[0] == "" {
		return "", nil
	}

	var result strings.Builder

	for _, file := range untrackedFiles {
		if file == "" {
			continue
		}

		// If a filter path is specified, check if this file matches
		if filterPath != "." {
			relPath, _ := filepath.Rel(workDir, filterPath)
			if !strings.HasPrefix(file, relPath) {
				continue
			}
		}

		// Read file content
		fullPath := filepath.Join(workDir, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		// Generate diff format for new file
		result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))
		result.WriteString("new file mode 100644\n")
		result.WriteString("index 0000000..0000000\n")
		result.WriteString("--- /dev/null\n")
		result.WriteString(fmt.Sprintf("+++ b/%s\n", file))
		result.WriteString("@@ -0,0 +1,")

		lines := strings.Split(string(content), "\n")
		result.WriteString(fmt.Sprintf("%d @@\n", len(lines)))

		for _, line := range lines {
			result.WriteString("+")
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}
