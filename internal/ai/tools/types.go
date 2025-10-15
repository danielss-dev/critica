package tools

import "context"

// ToolInfo describes a tool's capabilities
type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

// ToolCall represents a call to a tool
type ToolCall struct {
	ID    string
	Name  string
	Input string
}

// ToolResponse represents a tool's response
type ToolResponse struct {
	Content string
	IsError bool
	Data    map[string]interface{}
}

// BaseTool is the interface all tools must implement
type BaseTool interface {
	Info() ToolInfo
	Run(ctx context.Context, call ToolCall) (ToolResponse, error)
}

// NewTextResponse creates a successful text response
func NewTextResponse(content string) ToolResponse {
	return ToolResponse{
		Content: content,
		IsError: false,
		Data:    make(map[string]interface{}),
	}
}

// NewTextErrorResponse creates an error text response
func NewTextErrorResponse(content string) ToolResponse {
	return ToolResponse{
		Content: content,
		IsError: true,
		Data:    make(map[string]interface{}),
	}
}

// Suggestion represents a code improvement suggestion
type Suggestion struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Type        string  `json:"type"` // "improvement", "bug_fix", "optimization"
	Confidence  float64 `json:"confidence"`
	Code        string  `json:"code,omitempty"`
	FilePath    string  `json:"file_path,omitempty"`
	LineNumber  int     `json:"line_number,omitempty"`
}

// GetContextValues extracts session and other context values
func GetContextValues(ctx context.Context) (sessionID string, userID string) {
	if val := ctx.Value("session_id"); val != nil {
		sessionID = val.(string)
	}
	if val := ctx.Value("user_id"); val != nil {
		userID = val.(string)
	}
	return
}
