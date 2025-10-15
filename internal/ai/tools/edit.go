package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielss-dev/critica/internal/ai/permission"
)

// EditParams defines parameters for the edit tool
type EditParams struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// EditTool safely edits files with permission checks
type EditTool struct {
	permissions permission.Service
}

// NewEditTool creates a new edit tool
func NewEditTool(permissions permission.Service) BaseTool {
	return &EditTool{
		permissions: permissions,
	}
}

// Info returns tool information
func (t *EditTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "edit_file",
		Description: "Safely edit files with permission checks",
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to the file to edit",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "Text to replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "Replacement text",
			},
		},
		Required: []string{"file_path", "old_string", "new_string"},
	}
}

// Run executes the edit tool
func (t *EditTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params EditParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	// Request permission
	sessionID, _ := GetContextValues(ctx)
	if !t.permissions.Request(permission.PermissionRequest{
		SessionID:   sessionID,
		ToolName:    "edit_file",
		Action:      "write",
		Description: fmt.Sprintf("Edit file %s", params.FilePath),
		Path:        filepath.Dir(params.FilePath),
	}) {
		return NewTextErrorResponse("permission denied"), nil
	}

	// Read current file content
	content, err := os.ReadFile(params.FilePath)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	// Perform the replacement
	oldContent := string(content)
	newContent := strings.Replace(oldContent, params.OldString, params.NewString, 1)

	if oldContent == newContent {
		return NewTextErrorResponse("no changes made - old_string not found"), nil
	}

	// Write the new content
	if err := os.WriteFile(params.FilePath, []byte(newContent), 0644); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to write file: %v", err)), nil
	}

	return NewTextResponse(fmt.Sprintf("Successfully edited %s", params.FilePath)), nil
}
