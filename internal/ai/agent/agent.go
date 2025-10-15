package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/danielss-dev/critica/internal/ai/llm"
	"github.com/danielss-dev/critica/internal/ai/llm/prompt"
	"github.com/danielss-dev/critica/internal/ai/permission"
	"github.com/danielss-dev/critica/internal/ai/session"
	"github.com/danielss-dev/critica/internal/ai/tools"
)

// AgentEventType represents the type of agent event
type AgentEventType string

const (
	AgentEventTypeError       AgentEventType = "error"
	AgentEventTypeResponse    AgentEventType = "response"
	AgentEventTypeAnalysis    AgentEventType = "analysis"
	AgentEventTypeSuggestion  AgentEventType = "suggestion"
	AgentEventTypeExplanation AgentEventType = "explanation"
)

// AgentEvent represents an event from the agent
type AgentEvent struct {
	Type    AgentEventType
	Content string
	Error   error
	Data    map[string]interface{}
}

// Service is the main agent service interface
type Service interface {
	AnalyzeDiff(ctx context.Context, diffContent string) (<-chan AgentEvent, error)
	SuggestImprovements(ctx context.Context, filePath string, diffContent string) (<-chan AgentEvent, error)
	ExplainChanges(ctx context.Context, diffContent string) (<-chan AgentEvent, error)
	ApplySuggestions(ctx context.Context, suggestions []tools.Suggestion) error
	Cancel(sessionID string)
}

type agent struct {
	llmProvider    llm.Provider
	tools          []tools.BaseTool
	permissions    permission.Service
	sessions       session.Service
	activeRequests sync.Map
}

// NewAgent creates a new agent service
func NewAgent(provider llm.Provider, permissions permission.Service, sessions session.Service) Service {
	return &agent{
		llmProvider: provider,
		tools:       getCriticaTools(permissions),
		permissions: permissions,
		sessions:    sessions,
	}
}

// AnalyzeDiff analyzes git diff content
func (a *agent) AnalyzeDiff(ctx context.Context, diffContent string) (<-chan AgentEvent, error) {
	events := make(chan AgentEvent)

	go func() {
		defer close(events)

		// Create analysis prompt
		promptText := prompt.DiffAnalysisPrompt(diffContent)

		// Get AI response
		response, err := a.llmProvider.Generate(ctx, promptText)
		if err != nil {
			events <- AgentEvent{Type: AgentEventTypeError, Error: err}
			return
		}

		events <- AgentEvent{
			Type:    AgentEventTypeAnalysis,
			Content: response,
			Data:    map[string]interface{}{"type": "diff_analysis"},
		}
	}()

	return events, nil
}

// SuggestImprovements generates improvement suggestions
func (a *agent) SuggestImprovements(ctx context.Context, filePath string, diffContent string) (<-chan AgentEvent, error) {
	events := make(chan AgentEvent)

	go func() {
		defer close(events)

		// Create suggestion prompt
		promptText := prompt.SuggestImprovementsPrompt(filePath, []string{diffContent})

		// Get AI response
		response, err := a.llmProvider.Generate(ctx, promptText)
		if err != nil {
			events <- AgentEvent{Type: AgentEventTypeError, Error: err}
			return
		}

		events <- AgentEvent{
			Type:    AgentEventTypeSuggestion,
			Content: response,
			Data:    map[string]interface{}{"file_path": filePath},
		}
	}()

	return events, nil
}

// ExplainChanges explains what the changes do
func (a *agent) ExplainChanges(ctx context.Context, diffContent string) (<-chan AgentEvent, error) {
	events := make(chan AgentEvent)

	go func() {
		defer close(events)

		// Create explanation prompt
		promptText := prompt.ExplainChangesPrompt(diffContent)

		// Get AI response
		response, err := a.llmProvider.Generate(ctx, promptText)
		if err != nil {
			events <- AgentEvent{Type: AgentEventTypeError, Error: err}
			return
		}

		events <- AgentEvent{
			Type:    AgentEventTypeExplanation,
			Content: response,
			Data:    map[string]interface{}{"type": "explanation"},
		}
	}()

	return events, nil
}

// ApplySuggestions applies suggested changes
func (a *agent) ApplySuggestions(ctx context.Context, suggestions []tools.Suggestion) error {
	// This would use the edit tool to apply suggestions
	// For now, just validate permissions
	for _, suggestion := range suggestions {
		if suggestion.FilePath != "" {
			sessionID, _ := tools.GetContextValues(ctx)
			if !a.permissions.Request(permission.PermissionRequest{
				SessionID:   sessionID,
				ToolName:    "apply_suggestion",
				Action:      "write",
				Description: fmt.Sprintf("Apply suggestion: %s", suggestion.Title),
				Path:        suggestion.FilePath,
			}) {
				return fmt.Errorf("permission denied for %s", suggestion.FilePath)
			}
		}
	}
	return nil
}

// Cancel cancels an active request
func (a *agent) Cancel(sessionID string) {
	a.sessions.Close(sessionID)
}

// getCriticaTools returns the list of available tools
func getCriticaTools(permissions permission.Service) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewAnalyzeTool(),
		tools.NewSuggestTool(),
		tools.NewExplainTool(),
		tools.NewEditTool(permissions),
	}
}
