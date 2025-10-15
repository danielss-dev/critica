package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielss-dev/critica/internal/ai/agent"
)

// AIChatModel represents the AI chat interface
type AIChatModel struct {
	viewport      viewport.Model
	textarea      textarea.Model
	messages      []ChatMessage
	ready         bool
	width         int
	height        int
	agentSvc      agent.Service
	diffContent   string
	contextMsg    string // Context message about what's being analyzed
	initialAction string // Initial AI action to execute (analyze, suggest, explain, chat)
	spinner       spinner.Model
	isProcessing  bool // Track when AI is processing
}

// ChatMessage represents a single chat message
type ChatMessage struct {
	Content string
	Type    string // "user", "ai", "system", "error"
	Time    time.Time
}

// AIResponseMsg is sent when AI responds
type AIResponseMsg struct {
	Content string
	IsError bool
}

// NewAIChat creates a new AI chat model
func NewAIChat(agentSvc agent.Service, diffContent string, contextMsg string) *AIChatModel {
	return NewAIChatWithAction(agentSvc, diffContent, contextMsg, "")
}

// NewAIChatWithAction creates a new AI chat model with an initial action
func NewAIChatWithAction(agentSvc agent.Service, diffContent string, contextMsg string, initialAction string) *AIChatModel {
	ta := textarea.New()
	ta.Placeholder = "Ask AI about the changes..."
	ta.Focus()
	ta.CharLimit = 500
	ta.SetWidth(50)
	ta.SetHeight(3)

	vp := viewport.New(50, 20)

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Pulse

	chat := &AIChatModel{
		textarea:      ta,
		viewport:      vp,
		messages:      []ChatMessage{},
		agentSvc:      agentSvc,
		diffContent:   diffContent,
		contextMsg:    contextMsg,
		initialAction: initialAction,
		spinner:       s,
		isProcessing:  false,
	}

	// Add initial context message
	if contextMsg != "" {
		chat.messages = append(chat.messages, ChatMessage{
			Content: contextMsg,
			Type:    "system",
			Time:    time.Now(),
		})
	}

	return chat
}

// Init initializes the AI chat
func (m AIChatModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

// SetSize implements the Sizeable interface
func (m *AIChatModel) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width)
	m.viewport.Width = width
	m.viewport.Height = height - 5
	m.ready = true
	m.updateViewport()

	// Execute initial action if specified
	if m.initialAction != "" {
		return m.executeInitialAction()
	}

	return nil
}

// GetSize implements the Sizeable interface
func (m *AIChatModel) GetSize() (int, int) {
	return m.width, m.height
}

// Update handles messages
func (m AIChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Fallback for direct usage (not through container)
		if !m.ready {
			m.width = msg.Width
			m.height = msg.Height
			m.textarea.SetWidth(msg.Width)
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 5
			m.ready = true
			m.updateViewport()
		}

	case AIResponseMsg:
		msgType := "ai"
		if msg.IsError {
			msgType = "error"
		}
		m.messages = append(m.messages, ChatMessage{
			Content: msg.Content,
			Type:    msgType,
			Time:    time.Now(),
		})
		m.isProcessing = false // AI response received, stop processing
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.textarea.Focused() {
				content := m.textarea.Value()
				if strings.TrimSpace(content) != "" {
					m.messages = append(m.messages, ChatMessage{
						Content: content,
						Type:    "user",
						Time:    time.Now(),
					})
					m.textarea.Reset()
					m.updateViewport()

					// Send to AI agent
					return m, m.sendToAI(content)
				}
			}
		case "esc":
			m.textarea.Blur()
		}
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

// working returns the status message when AI is processing
func (m AIChatModel) working() string {
	if !m.isProcessing {
		return ""
	}

	status := "Thinking..."
	if m.agentSvc == nil {
		status = "Waiting for AI service..."
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#58a6ff")).
		Bold(true)

	return style.Render(fmt.Sprintf("%s %s", m.spinner.View(), status))
}

// View renders the AI chat
func (m AIChatModel) View() string {
	if !m.ready {
		return "Initializing AI chat..."
	}

	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#58a6ff")).
		Render("AI Code Assistant")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Chat messages
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	// Working status
	workingStatus := m.working()
	if workingStatus != "" {
		b.WriteString(workingStatus)
		b.WriteString("\n\n")
	}

	// Input area
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	// Help
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Enter: send | Esc: blur input | Ctrl+C: quit")
	b.WriteString(help)

	return b.String()
}

// sendToAI sends a message to the AI agent
func (m *AIChatModel) sendToAI(message string) tea.Cmd {
	// Set processing state
	m.isProcessing = true

	return func() tea.Msg {
		// Check if agent service is available
		if m.agentSvc == nil {
			return AIResponseMsg{Content: "Error: AI agent service is not available. Please check your configuration.", IsError: true}
		}

		// Determine the type of request based on keywords
		lowerMsg := strings.ToLower(message)

		var events <-chan agent.AgentEvent
		var err error
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if strings.Contains(lowerMsg, "analyze") || strings.Contains(lowerMsg, "review") {
			events, err = m.agentSvc.AnalyzeDiff(ctx, m.diffContent)
		} else if strings.Contains(lowerMsg, "suggest") || strings.Contains(lowerMsg, "improve") {
			events, err = m.agentSvc.SuggestImprovements(ctx, "", m.diffContent)
		} else if strings.Contains(lowerMsg, "explain") {
			events, err = m.agentSvc.ExplainChanges(ctx, m.diffContent)
		} else {
			// Default to explanation
			events, err = m.agentSvc.ExplainChanges(ctx, m.diffContent)
		}

		if err != nil {
			return AIResponseMsg{Content: "Error: " + err.Error(), IsError: true}
		}

		// Collect response with timeout handling
		var response strings.Builder
		timeout := time.After(30 * time.Second)

		for {
			select {
			case event, ok := <-events:
				if !ok {
					// Channel closed
					if response.Len() == 0 {
						return AIResponseMsg{Content: "No response received from AI agent. The service may be unavailable.", IsError: true}
					}
					return AIResponseMsg{Content: response.String(), IsError: false}
				}

				if event.Error != nil {
					return AIResponseMsg{Content: "Error: " + event.Error.Error(), IsError: true}
				}
				response.WriteString(event.Content)

			case <-timeout:
				if response.Len() == 0 {
					return AIResponseMsg{Content: "Timeout: AI agent did not respond within 30 seconds. Please check your internet connection and API key.", IsError: true}
				}
				return AIResponseMsg{Content: response.String(), IsError: false}
			}
		}
	}
}

// executeInitialAction executes the initial AI action
func (m *AIChatModel) executeInitialAction() tea.Cmd {
	// Add a message indicating the action is being executed
	actionMsg := "Executing " + m.initialAction + "..."
	m.messages = append(m.messages, ChatMessage{
		Content: actionMsg,
		Type:    "system",
		Time:    time.Now(),
	})
	m.updateViewport()

	// Check if agent service is available
	if m.agentSvc == nil {
		m.messages = append(m.messages, ChatMessage{
			Content: "Error: AI agent service is not available. Please check your configuration.",
			Type:    "error",
			Time:    time.Now(),
		})
		m.updateViewport()
		return nil
	}

	// Add a test message to verify the service is working
	m.messages = append(m.messages, ChatMessage{
		Content: "Testing AI connection...",
		Type:    "system",
		Time:    time.Now(),
	})
	m.updateViewport()

	// For now, let's add a simple test response to verify the chat is working
	m.messages = append(m.messages, ChatMessage{
		Content: "AI service is available. Processing your request...",
		Type:    "ai",
		Time:    time.Now(),
	})
	m.updateViewport()

	// Execute the appropriate action based on the initial action
	switch m.initialAction {
	case "analyze":
		return m.sendToAI("Please analyze the code changes for issues and quality.")
	case "suggest":
		return m.sendToAI("Please suggest improvements for the code changes.")
	case "explain":
		return m.sendToAI("Please explain what these code changes do.")
	case "chat":
		// For chat, just show a welcome message
		m.messages = append(m.messages, ChatMessage{
			Content: "AI chat ready. You can ask any questions about the code changes.",
			Type:    "ai",
			Time:    time.Now(),
		})
		m.updateViewport()
		return nil
	default:
		// Default to explanation
		return m.sendToAI("Please explain what these code changes do.")
	}
}

// updateViewport updates the viewport content with all messages
func (m *AIChatModel) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		var style lipgloss.Style
		prefix := ""

		switch msg.Type {
		case "user":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff"))
			prefix = "You: "
		case "ai":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ee787"))
			prefix = "AI: "
		case "error":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149"))
			prefix = "Error: "
		case "system":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
			prefix = "System: "
		}

		content.WriteString(style.Render(prefix + msg.Content))
		content.WriteString("\n\n")
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}
