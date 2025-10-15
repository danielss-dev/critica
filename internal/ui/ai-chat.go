package ui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielss-dev/critica/internal/ai/agent"
)

// AIChatModel represents the AI chat interface
type AIChatModel struct {
	viewport    viewport.Model
	textarea    textarea.Model
	messages    []ChatMessage
	ready       bool
	width       int
	height      int
	agentSvc    agent.Service
	diffContent string
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
func NewAIChat(agentSvc agent.Service, diffContent string) *AIChatModel {
	ta := textarea.New()
	ta.Placeholder = "Ask AI about the changes..."
	ta.Focus()
	ta.CharLimit = 500
	ta.SetWidth(50)
	ta.SetHeight(3)

	vp := viewport.New(50, 20)

	return &AIChatModel{
		textarea:    ta,
		viewport:    vp,
		messages:    []ChatMessage{},
		agentSvc:    agentSvc,
		diffContent: diffContent,
	}
}

// Init initializes the AI chat
func (m AIChatModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages
func (m AIChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width)
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 5
		m.ready = true

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

	return m, tea.Batch(tiCmd, vpCmd)
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
	return func() tea.Msg {
		// Determine the type of request based on keywords
		lowerMsg := strings.ToLower(message)

		var events <-chan agent.AgentEvent
		var err error
		ctx := context.TODO()

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

		// Collect response
		var response strings.Builder
		for event := range events {
			if event.Error != nil {
				return AIResponseMsg{Content: "Error: " + event.Error.Error(), IsError: true}
			}
			response.WriteString(event.Content)
		}

		return AIResponseMsg{Content: response.String(), IsError: false}
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
