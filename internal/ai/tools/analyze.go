package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danielss-dev/critica/internal/parser"
)

// AnalyzeParams defines parameters for the analyze tool
type AnalyzeParams struct {
	DiffContent string `json:"diff_content"`
	FilePath    string `json:"file_path,omitempty"`
}

// AnalyzeTool analyzes git diffs for code quality and issues
type AnalyzeTool struct{}

// NewAnalyzeTool creates a new analyze tool
func NewAnalyzeTool() BaseTool {
	return &AnalyzeTool{}
}

// Info returns tool information
func (t *AnalyzeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "analyze_diff",
		Description: "Analyze git diff content for code quality, potential issues, and improvements",
		Parameters: map[string]any{
			"diff_content": map[string]any{
				"type":        "string",
				"description": "The git diff content to analyze",
			},
			"file_path": map[string]any{
				"type":        "string",
				"description": "Optional file path for context",
			},
		},
		Required: []string{"diff_content"},
	}
}

// Run executes the analyze tool
func (t *AnalyzeTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params AnalyzeParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	// Parse the diff
	files, err := parser.ParseDiff(params.DiffContent)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to parse diff: %v", err)), nil
	}

	// Analyze each file
	var analysis strings.Builder
	analysis.WriteString("# Diff Analysis\n\n")

	for _, file := range files {
		analysis.WriteString(fmt.Sprintf("## File: %s\n", file.NewPath))
		analysis.WriteString(fmt.Sprintf("**Status:** %s\n", getFileStatus(file)))
		analysis.WriteString(fmt.Sprintf("**Changes:** %d additions, %d deletions\n\n",
			countAdditions(file), countDeletions(file)))

		// Analyze hunks for potential issues
		issues := analyzeFile(file)
		if len(issues) > 0 {
			analysis.WriteString("**Potential Issues:**\n")
			for _, issue := range issues {
				analysis.WriteString(fmt.Sprintf("- %s\n", issue))
			}
			analysis.WriteString("\n")
		}
	}

	return NewTextResponse(analysis.String()), nil
}

func getFileStatus(file parser.FileDiff) string {
	if file.IsNew {
		return "new file"
	} else if file.IsDeleted {
		return "deleted"
	} else if file.IsRenamed {
		return "renamed"
	}
	return "modified"
}

func countAdditions(file parser.FileDiff) int {
	count := 0
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == parser.LineAdded {
				count++
			}
		}
	}
	return count
}

func countDeletions(file parser.FileDiff) int {
	count := 0
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == parser.LineDeleted {
				count++
			}
		}
	}
	return count
}

func analyzeFile(file parser.FileDiff) []string {
	var issues []string

	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == parser.LineAdded {
				// Check for common issues
				content := strings.TrimSpace(line.Content)

				// Debug statements
				if strings.Contains(content, "console.log") ||
					strings.Contains(content, "fmt.Println") ||
					strings.Contains(content, "print(") {
					issues = append(issues, fmt.Sprintf("Line %d: Debug statement detected", line.NewLineNum))
				}

				// TODO comments
				if strings.Contains(content, "TODO") || strings.Contains(content, "FIXME") {
					issues = append(issues, fmt.Sprintf("Line %d: TODO/FIXME comment found", line.NewLineNum))
				}

				// Panic usage
				if strings.Contains(content, "panic(") {
					issues = append(issues, fmt.Sprintf("Line %d: Consider proper error handling instead of panic", line.NewLineNum))
				}

				// Long lines (over 120 characters)
				if len(content) > 120 {
					issues = append(issues, fmt.Sprintf("Line %d: Line exceeds 120 characters (%d chars)", line.NewLineNum, len(content)))
				}
			}
		}
	}

	return issues
}
