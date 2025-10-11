package ui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielss-dev/critica/internal/parser"
)

type viewMode int

const (
	fileListView viewMode = iota
	diffView
)

type model struct {
	files       []parser.FileDiff
	fileItems   []list.Item
	list        list.Model
	textInput   textinput.Model
	viewMode    viewMode
	selectedIdx int
	collapsed   map[int]bool
	useColor    bool
	unified     bool
	width       int
	height      int
	renderer    *Renderer
}

type fileItem struct {
	name   string
	status string
	index  int
}

func (f fileItem) FilterValue() string { return f.name }
func (f fileItem) Title() string       { return f.name }
func (f fileItem) Description() string { return f.status }

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Back       key.Binding
	Quit       key.Binding
	Search     key.Binding
	ToggleView key.Binding
	Collapse   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select file"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back to file list"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search files"),
	),
	ToggleView: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle split/unified"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "collapse/expand file"),
	),
}

func RunInteractive(files []parser.FileDiff, useColor, unified bool) error {
	// Create file items for list
	items := make([]list.Item, len(files))
	for i, file := range files {
		status := "modified"
		if file.IsNew {
			status = "new file"
		} else if file.IsDeleted {
			status = "deleted"
		} else if file.IsRenamed {
			status = "renamed"
		}
		items[i] = fileItem{
			name:   file.NewPath,
			status: status,
			index:  i,
		}
	}

	// Create list
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Changed Files (press ? for help)"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.KeyMap.Quit.Unbind()

	// Create text input for search
	ti := textinput.New()
	ti.Placeholder = "Search files..."
	ti.CharLimit = 50

	m := model{
		files:     files,
		fileItems: items,
		list:      l,
		textInput: ti,
		viewMode:  fileListView,
		collapsed: make(map[int]bool),
		useColor:  useColor,
		unified:   unified,
		renderer:  NewRenderer(useColor, unified),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		switch m.viewMode {
		case fileListView:
			switch {
			case key.Matches(msg, keys.Quit):
				return m, tea.Quit

			case key.Matches(msg, keys.Enter):
				if len(m.list.Items()) > 0 {
					item := m.list.SelectedItem().(fileItem)
					m.selectedIdx = item.index
					m.viewMode = diffView
					return m, nil
				}

			case key.Matches(msg, keys.ToggleView):
				m.unified = !m.unified
				m.renderer.unified = m.unified
				return m, nil

			default:
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}

		case diffView:
			switch {
			case key.Matches(msg, keys.Quit):
				return m, tea.Quit

			case key.Matches(msg, keys.Back):
				m.viewMode = fileListView
				return m, nil

			case key.Matches(msg, keys.ToggleView):
				m.unified = !m.unified
				m.renderer.unified = m.unified
				return m, nil

			case key.Matches(msg, keys.Collapse):
				m.collapsed[m.selectedIdx] = !m.collapsed[m.selectedIdx]
				return m, nil

			case key.Matches(msg, keys.Up):
				if m.selectedIdx > 0 {
					m.selectedIdx--
				}
				return m, nil

			case key.Matches(msg, keys.Down):
				if m.selectedIdx < len(m.files)-1 {
					m.selectedIdx++
				}
				return m, nil
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	switch m.viewMode {
	case fileListView:
		return m.renderFileList()
	case diffView:
		return m.renderDiff()
	default:
		return ""
	}
}

func (m model) renderFileList() string {
	var b strings.Builder

	b.WriteString(m.list.View())
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	viewMode := "split-screen"
	if m.unified {
		viewMode = "unified"
	}

	help := fmt.Sprintf("Current view: %s | Press 'tab' to toggle | Press 'enter' to view diff | Press 'q' to quit", viewMode)
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) renderDiff() string {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.files) {
		return "No file selected"
	}

	file := m.files[m.selectedIdx]

	var b strings.Builder

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	viewMode := "Split View"
	if m.unified {
		viewMode = "Unified View"
	}

	title := fmt.Sprintf("%s (%d/%d) - %s", file.NewPath, m.selectedIdx+1, len(m.files), viewMode)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Check if collapsed
	if m.collapsed[m.selectedIdx] {
		collapsedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		b.WriteString(collapsedStyle.Render(fmt.Sprintf("File collapsed. Press 'space' to expand.")))
	} else {
		// Render file diff
		var diffOutput strings.Builder

		// Temporarily capture stdout
		oldStdout := fmt.Sprint
		defer func() { _ = oldStdout }()

		// Render hunks
		lexer := m.renderer.getLexer(file.Extension)
		for _, hunk := range file.Hunks {
			if m.unified {
				for _, line := range hunk.Lines {
					diffOutput.WriteString(m.renderLineDirect(line, lexer))
					diffOutput.WriteString("\n")
				}
			} else {
				diffOutput.WriteString(m.renderHunkSplit(hunk, lexer))
			}
		}

		b.WriteString(diffOutput.String())
	}

	b.WriteString("\n\n")

	// Help bar
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := "↑/↓: navigate files | space: collapse/expand | tab: toggle view | esc: back | q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) renderLineDirect(line parser.Line, lexer chroma.Lexer) string {
	var lineNum int
	var prefix string

	switch line.Type {
	case parser.LineDeleted:
		lineNum = line.OldLineNum
		prefix = "-"
	case parser.LineAdded:
		lineNum = line.NewLineNum
		prefix = "+"
	case parser.LineUnchanged:
		lineNum = line.NewLineNum
		prefix = " "
	}

	lineNumStr := ""
	if lineNum > 0 {
		lineNumStr = fmt.Sprintf("%4d", lineNum)
	} else {
		lineNumStr = "    "
	}

	content := line.Content

	var lineStyle lipgloss.Style
	if m.useColor {
		switch line.Type {
		case parser.LineDeleted:
			lineStyle = m.renderer.theme.DeletedLineStyle
		case parser.LineAdded:
			lineStyle = m.renderer.theme.AddedLineStyle
		case parser.LineUnchanged:
			lineStyle = m.renderer.theme.UnchangedLineStyle
		}
	} else {
		lineStyle = lipgloss.NewStyle()
	}

	fullLine := lineNumStr + " " + prefix + " " + content

	if m.useColor {
		return lineStyle.Render(fullLine)
	}
	return fullLine
}

func (m model) renderHunkSplit(hunk parser.Hunk, lexer chroma.Lexer) string {
	var b strings.Builder

	columnWidth := (m.width - 3) / 2
	if columnWidth < 40 {
		columnWidth = 40
	}

	for _, line := range hunk.Lines {
		leftLine := ""
		rightLine := ""

		switch line.Type {
		case parser.LineDeleted:
			leftLine = m.renderer.formatLine(line, columnWidth, lexer, true)
			rightLine = m.renderer.formatEmptyLine(columnWidth)
		case parser.LineAdded:
			leftLine = m.renderer.formatEmptyLine(columnWidth)
			rightLine = m.renderer.formatLine(line, columnWidth, lexer, false)
		case parser.LineUnchanged:
			leftLine = m.renderer.formatLine(line, columnWidth, lexer, true)
			rightLine = m.renderer.formatLine(line, columnWidth, lexer, false)
		}

		separator := m.renderer.theme.SeparatorStyle.Render("│")
		b.WriteString(fmt.Sprintf("%s %s %s\n", leftLine, separator, rightLine))
	}

	return b.String()
}
