package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LineType represents the type of a line in a diff
type LineType int

const (
	LineUnchanged LineType = iota
	LineAdded
	LineDeleted
	LineContext
)

// Line represents a single line in a diff
type Line struct {
	Type       LineType
	Content    string
	OldLineNum int // 0 if not applicable
	NewLineNum int // 0 if not applicable
}

// Hunk represents a chunk of changes in a file
type Hunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []Line
}

// FileDiff represents all changes in a single file
type FileDiff struct {
	OldPath    string
	NewPath    string
	IsNew      bool
	IsDeleted  bool
	IsRenamed  bool
	Extension  string
	Hunks      []Hunk
}

var (
	diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	filePathRegex   = regexp.MustCompile(`^[+-]{3} (.+)$`)
	hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
)

// ParseDiff parses git diff output into structured FileDiff objects
func ParseDiff(diffOutput string) ([]FileDiff, error) {
	if diffOutput == "" {
		return []FileDiff{}, nil
	}

	lines := strings.Split(diffOutput, "\n")
	var files []FileDiff
	var currentFile *FileDiff
	var currentHunk *Hunk
	var oldLineNum, newLineNum int

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for diff header (start of new file)
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous file if exists
			if currentFile != nil {
				files = append(files, *currentFile)
			}

			// Start new file
			currentFile = &FileDiff{
				OldPath: matches[1],
				NewPath: matches[2],
			}
			currentFile.Extension = filepath.Ext(currentFile.NewPath)
			currentHunk = nil
			continue
		}

		if currentFile == nil {
			continue
		}

		// Check for file status indicators
		if strings.HasPrefix(line, "new file mode") {
			currentFile.IsNew = true
			continue
		}
		if strings.HasPrefix(line, "deleted file mode") {
			currentFile.IsDeleted = true
			continue
		}
		if strings.HasPrefix(line, "rename from") {
			currentFile.IsRenamed = true
			continue
		}

		// Skip index lines, file mode lines
		if strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "Binary files") ||
			strings.HasPrefix(line, "similarity index") ||
			strings.HasPrefix(line, "rename to") {
			continue
		}

		// Skip --- and +++ lines (we already have paths)
		if matches := filePathRegex.FindStringSubmatch(line); matches != nil {
			continue
		}

		// Check for hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous hunk
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			// Parse hunk information
			oldStart, _ := strconv.Atoi(matches[1])
			oldLines := 1
			if matches[2] != "" {
				oldLines, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newLines := 1
			if matches[4] != "" {
				newLines, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldLines: oldLines,
				NewStart: newStart,
				NewLines: newLines,
				Lines:    []Line{},
			}

			oldLineNum = oldStart
			newLineNum = newStart
			continue
		}

		// Parse diff lines
		if currentHunk != nil {
			if len(line) == 0 {
				continue
			}

			prefix := line[0]
			content := ""
			if len(line) > 1 {
				content = line[1:]
			}

			switch prefix {
			case '+':
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:       LineAdded,
					Content:    content,
					OldLineNum: 0,
					NewLineNum: newLineNum,
				})
				newLineNum++

			case '-':
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:       LineDeleted,
					Content:    content,
					OldLineNum: oldLineNum,
					NewLineNum: 0,
				})
				oldLineNum++

			case ' ':
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:       LineUnchanged,
					Content:    content,
					OldLineNum: oldLineNum,
					NewLineNum: newLineNum,
				})
				oldLineNum++
				newLineNum++

			case '\\':
				// "\ No newline at end of file" - skip
				continue
			}
		}
	}

	// Save last hunk and file
	if currentHunk != nil && currentFile != nil {
		currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
	}
	if currentFile != nil {
		files = append(files, *currentFile)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no diff data parsed")
	}

	return files, nil
}
