package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danielss-dev/critica/internal/parser"
)

// SuggestParams defines parameters for the suggest tool
type SuggestParams struct {
	FilePath    string `json:"file_path"`
	DiffContent string `json:"diff_content"`
	Context     string `json:"context,omitempty"`
}

// SuggestTool generates code improvement suggestions
type SuggestTool struct{}

// NewSuggestTool creates a new suggest tool
func NewSuggestTool() BaseTool {
	return &SuggestTool{}
}

// Info returns tool information
func (t *SuggestTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "suggest_improvements",
		Description: "Generate code improvement suggestions based on diff content",
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to the file being analyzed",
			},
			"diff_content": map[string]any{
				"type":        "string",
				"description": "The git diff content to analyze",
			},
			"context": map[string]any{
				"type":        "string",
				"description": "Optional additional context",
			},
		},
		Required: []string{"file_path", "diff_content"},
	}
}

// Run executes the suggest tool
func (t *SuggestTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params SuggestParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	// Parse the diff
	files, err := parser.ParseDiff(params.DiffContent)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to parse diff: %v", err)), nil
	}

	// Analyze and generate suggestions
	suggestions := t.analyzeAndSuggest(params.FilePath, files)

	// Format suggestions as JSON
	suggestionsJSON, err := json.Marshal(suggestions)
	if err != nil {
		return NewTextErrorResponse("failed to marshal suggestions"), nil
	}

	return NewTextResponse(string(suggestionsJSON)), nil
}

func (t *SuggestTool) analyzeAndSuggest(filePath string, files []parser.FileDiff) []Suggestion {
	var suggestions []Suggestion

	for _, file := range files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				if line.Type == parser.LineAdded {
					if suggestion := t.analyzeLine(file.NewPath, line); suggestion != nil {
						suggestions = append(suggestions, *suggestion)
					}
				}
			}
		}
	}

	return suggestions
}

func (t *SuggestTool) analyzeLine(filePath string, line parser.Line) *Suggestion {
	content := strings.TrimSpace(line.Content)

	// Debug print statements
	if strings.Contains(content, "fmt.Println") || strings.Contains(content, "console.log") {
		return &Suggestion{
			Title:       "Remove debug print statement",
			Description: "Debug print statements should be removed before committing code. Consider using proper logging instead.",
			Type:        "improvement",
			Confidence:  0.9,
			FilePath:    filePath,
			LineNumber:  line.NewLineNum,
		}
	}

	// Panic usage
	if strings.Contains(content, "panic(") {
		return &Suggestion{
			Title:       "Replace panic with error handling",
			Description: "Using panic should be avoided in most cases. Consider returning an error instead for more graceful error handling.",
			Type:        "bug_fix",
			Confidence:  0.85,
			FilePath:    filePath,
			LineNumber:  line.NewLineNum,
			Code:        "return fmt.Errorf(\"...\")",
		}
	}

	// Empty error checks
	if strings.Contains(content, "if err != nil {") {
		// Check if the next line might be empty or just returns
		// This is a simple heuristic
		return &Suggestion{
			Title:       "Review error handling",
			Description: "Ensure error handling is comprehensive and provides useful context.",
			Type:        "improvement",
			Confidence:  0.6,
			FilePath:    filePath,
			LineNumber:  line.NewLineNum,
		}
	}

	// TODO/FIXME comments
	if strings.Contains(content, "TODO") || strings.Contains(content, "FIXME") {
		return &Suggestion{
			Title:       "Address TODO/FIXME comment",
			Description: "TODO/FIXME comments indicate incomplete work. Consider addressing these before merging.",
			Type:        "improvement",
			Confidence:  0.7,
			FilePath:    filePath,
			LineNumber:  line.NewLineNum,
		}
	}

	// Long lines
	if len(content) > 120 {
		return &Suggestion{
			Title:       "Break long line",
			Description: fmt.Sprintf("Line is %d characters long. Consider breaking it into multiple lines for better readability.", len(content)),
			Type:        "improvement",
			Confidence:  0.75,
			FilePath:    filePath,
			LineNumber:  line.NewLineNum,
		}
	}

	return nil
}
