package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danielss-dev/critica/internal/parser"
)

// ExplainParams defines parameters for the explain tool
type ExplainParams struct {
	DiffContent string `json:"diff_content"`
}

// ExplainTool generates explanations for code changes
type ExplainTool struct{}

// NewExplainTool creates a new explain tool
func NewExplainTool() BaseTool {
	return &ExplainTool{}
}

// Info returns tool information
func (t *ExplainTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "explain_changes",
		Description: "Explain what code changes do in clear, simple language",
		Parameters: map[string]any{
			"diff_content": map[string]any{
				"type":        "string",
				"description": "The git diff content to explain",
			},
		},
		Required: []string{"diff_content"},
	}
}

// Run executes the explain tool
func (t *ExplainTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ExplainParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	// Parse the diff
	files, err := parser.ParseDiff(params.DiffContent)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to parse diff: %v", err)), nil
	}

	// Generate explanation
	explanation := t.explainChanges(files)

	return NewTextResponse(explanation), nil
}

func (t *ExplainTool) explainChanges(files []parser.FileDiff) string {
	var explanation strings.Builder

	explanation.WriteString("# Changes Explanation\n\n")

	if len(files) == 0 {
		return "No changes detected."
	}

	// Overview
	totalFiles := len(files)
	newFiles := 0
	deletedFiles := 0
	modifiedFiles := 0

	for _, file := range files {
		if file.IsNew {
			newFiles++
		} else if file.IsDeleted {
			deletedFiles++
		} else {
			modifiedFiles++
		}
	}

	explanation.WriteString("## Overview\n\n")
	explanation.WriteString(fmt.Sprintf("This change affects %d file(s):\n", totalFiles))
	if newFiles > 0 {
		explanation.WriteString(fmt.Sprintf("- %d new file(s)\n", newFiles))
	}
	if modifiedFiles > 0 {
		explanation.WriteString(fmt.Sprintf("- %d modified file(s)\n", modifiedFiles))
	}
	if deletedFiles > 0 {
		explanation.WriteString(fmt.Sprintf("- %d deleted file(s)\n", deletedFiles))
	}
	explanation.WriteString("\n")

	// Detailed explanations per file
	for _, file := range files {
		explanation.WriteString(fmt.Sprintf("## %s\n\n", file.NewPath))

		if file.IsNew {
			explanation.WriteString("**New file created**\n\n")
			explanation.WriteString(t.explainNewFile(file))
		} else if file.IsDeleted {
			explanation.WriteString("**File deleted**\n\n")
			explanation.WriteString("This file has been removed from the codebase.\n\n")
		} else {
			explanation.WriteString(t.explainModifiedFile(file))
		}
	}

	return explanation.String()
}

func (t *ExplainTool) explainNewFile(file parser.FileDiff) string {
	var explain strings.Builder

	additions := countAdditions(file)
	explain.WriteString(fmt.Sprintf("This new file contains %d lines of code.\n\n", additions))

	// Analyze what the file might be doing based on content
	hasImports := false
	hasFunctions := false
	hasStructs := false

	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			content := strings.TrimSpace(line.Content)
			if strings.HasPrefix(content, "import ") || strings.HasPrefix(content, "package ") {
				hasImports = true
			}
			if strings.HasPrefix(content, "func ") {
				hasFunctions = true
			}
			if strings.HasPrefix(content, "type ") && strings.Contains(content, "struct") {
				hasStructs = true
			}
		}
	}

	if hasImports {
		explain.WriteString("- Contains package/import declarations\n")
	}
	if hasStructs {
		explain.WriteString("- Defines new data structures\n")
	}
	if hasFunctions {
		explain.WriteString("- Implements new functions/methods\n")
	}

	return explain.String()
}

func (t *ExplainTool) explainModifiedFile(file parser.FileDiff) string {
	var explain strings.Builder

	additions := countAdditions(file)
	deletions := countDeletions(file)

	explain.WriteString(fmt.Sprintf("**Modified:** +%d lines, -%d lines\n\n", additions, deletions))

	// Analyze the nature of changes
	hasLogicChanges := false
	hasCommentChanges := false
	hasImportChanges := false

	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == parser.LineAdded || line.Type == parser.LineDeleted {
				content := strings.TrimSpace(line.Content)

				if strings.HasPrefix(content, "//") || strings.HasPrefix(content, "/*") {
					hasCommentChanges = true
				} else if strings.HasPrefix(content, "import ") {
					hasImportChanges = true
				} else if content != "" {
					hasLogicChanges = true
				}
			}
		}
	}

	explain.WriteString("**Changes include:**\n")
	if hasLogicChanges {
		explain.WriteString("- Code logic modifications\n")
	}
	if hasImportChanges {
		explain.WriteString("- Import/dependency updates\n")
	}
	if hasCommentChanges {
		explain.WriteString("- Documentation/comment updates\n")
	}

	return explain.String()
}
