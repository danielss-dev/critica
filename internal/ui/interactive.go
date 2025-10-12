package ui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
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
	searchMode
)

type model struct {
	files            []parser.FileDiff
	fileItems        []list.Item
	list             list.Model
	textInput        textinput.Model
	viewMode         viewMode
	selectedIdx      int
	collapsed        map[int]bool
	useColor         bool
	unified          bool
	width            int
	height           int
	renderer         *Renderer
	scrollOffset     int  // Current scroll position in diff view
	previewCollapsed bool // Whether the preview pane is collapsed
}

type fileItem struct {
	name   string
	status string
	index  int
}

func (f fileItem) FilterValue() string { return f.name }
func (f fileItem) Title() string       { return f.name }
func (f fileItem) Description() string { return f.status }

// Custom delegate with muted colors
type customDelegate struct {
	list.DefaultDelegate
}

func newCustomDelegate() customDelegate {
	d := customDelegate{DefaultDelegate: list.NewDefaultDelegate()}

	// Professional but vibrant colors for list items
	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#c9d1d9")). // Brighter gray-white
		Padding(0, 0, 0, 2)

	d.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8b949e")). // Medium gray
		Padding(0, 0, 0, 2)

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#58a6ff")). // Nice blue accent
		Foreground(lipgloss.Color("#ffffff")).       // White
		Padding(0, 0, 0, 1)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#58a6ff")). // Nice blue accent
		Foreground(lipgloss.Color("#79c0ff")).       // Light blue
		Padding(0, 0, 0, 1)

	d.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6e7681")).
		Padding(0, 0, 0, 2)

	d.Styles.DimmedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#484f58")).
		Padding(0, 0, 0, 2)

	return d
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

	// Create list with custom delegate
	delegate := newCustomDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Changed Files"

	// Customize list title style (professional but vibrant)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("#1f2937")). // Darker background
		Foreground(lipgloss.Color("#58a6ff")). // Nice blue
		Padding(0, 1).
		Bold(true)

	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.KeyMap.Quit.Unbind()

	// Create text input for search
	ti := textinput.New()
	ti.Placeholder = "Search files..."
	ti.CharLimit = 50

	// Initialize all files as expanded (not collapsed)
	collapsed := make(map[int]bool)
	for i := range files {
		collapsed[i] = false
	}

	m := model{
		files:            files,
		fileItems:        items,
		list:             l,
		textInput:        ti,
		viewMode:         fileListView,
		collapsed:        collapsed,
		useColor:         useColor,
		unified:          unified,
		renderer:         NewRenderer(useColor, unified),
		previewCollapsed: false, // Show preview by default
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
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "/":
				m.viewMode = searchMode
				m.textInput.Focus()
				m.textInput.SetValue("")
				return m, nil

			case " ":
				// Toggle preview collapse
				m.previewCollapsed = !m.previewCollapsed
				return m, nil

			case "o", "enter":
				// Open file in full diff view
				if len(m.list.Items()) > 0 {
					selectedItem := m.list.SelectedItem()
					if selectedItem != nil {
						if item, ok := selectedItem.(fileItem); ok {
							m.selectedIdx = item.index
							m.scrollOffset = 0
							m.viewMode = diffView
							return m, nil
						}
					}
				}
				return m, nil

			case "tab":
				m.unified = !m.unified
				m.renderer.unified = m.unified
				return m, nil

			default:
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}

		case searchMode:
			switch msg.String() {
			case "enter":
				// If there's a selected file, go to diff view
				if len(m.list.Items()) > 0 {
					selectedItem := m.list.SelectedItem()
					if selectedItem != nil {
						if item, ok := selectedItem.(fileItem); ok {
							m.selectedIdx = item.index
							m.scrollOffset = 0 // Reset scroll when entering diff view
							m.viewMode = diffView
							m.textInput.Blur()
							return m, nil
						}
					}
				}
				// Otherwise, return to file list view
				m.filterList()
				m.viewMode = fileListView
				m.textInput.Blur()
				return m, nil

			case "esc":
				// Cancel search and return to file list view
				m.viewMode = fileListView
				m.textInput.Blur()
				m.textInput.SetValue("")
				m.resetList()
				return m, nil

			case "up", "down":
				// Allow navigating filtered list
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd

			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				m.filterList()
				return m, cmd
			}

		case diffView:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc", "backspace":
				m.viewMode = fileListView
				return m, nil

			case "/":
				m.viewMode = searchMode
				m.textInput.Focus()
				m.textInput.SetValue("")
				return m, nil

			case "tab":
				m.unified = !m.unified
				m.renderer.unified = m.unified
				return m, nil

			case " ":
				m.collapsed[m.selectedIdx] = !m.collapsed[m.selectedIdx]
				return m, nil

			// Vim motions for scrolling within file
			case "j", "down":
				m.scrollOffset++
				return m, nil

			case "k", "up":
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
				return m, nil

			case "d", "ctrl+d":
				// Page down
				m.scrollOffset += m.height / 2
				return m, nil

			case "u", "ctrl+u":
				// Page up
				m.scrollOffset -= m.height / 2
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
				return m, nil

			case "g":
				// Go to top
				m.scrollOffset = 0
				return m, nil

			case "G":
				// Go to bottom (will be clamped in render)
				m.scrollOffset = 999999
				return m, nil

			// Vim motions for file navigation
			case "h", "left":
				if m.selectedIdx > 0 {
					m.selectedIdx--
					m.scrollOffset = 0 // Reset scroll when changing files
				}
				return m, nil

			case "l", "right":
				if m.selectedIdx < len(m.files)-1 {
					m.selectedIdx++
					m.scrollOffset = 0 // Reset scroll when changing files
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
	case searchMode:
		return m.renderSearch()
	default:
		return ""
	}
}

// filterList filters the file list based on the search input
func (m *model) filterList() {
	query := strings.ToLower(m.textInput.Value())
	if query == "" {
		m.resetList()
		return
	}

	var filteredItems []list.Item
	for _, item := range m.fileItems {
		fileItem := item.(fileItem)
		if strings.Contains(strings.ToLower(fileItem.name), query) {
			filteredItems = append(filteredItems, item)
		}
	}

	m.list.SetItems(filteredItems)
}

// resetList resets the file list to show all files
func (m *model) resetList() {
	m.list.SetItems(m.fileItems)
}

func (m model) renderFileList() string {
	if m.previewCollapsed {
		// Show only file list when preview is collapsed
		var b strings.Builder
		b.WriteString(m.list.View())
		b.WriteString("\n\n")

		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		help := "space: show preview | o/enter: open full view | /: search | tab: toggle view | q: quit"
		b.WriteString(helpStyle.Render(help))
		return b.String()
	}

	// Side-by-side layout: file list + preview
	var b strings.Builder

	// Calculate widths
	fileListWidth := 40
	if m.width < 120 {
		fileListWidth = m.width / 3
	}
	previewWidth := m.width - fileListWidth - 3 // -3 for separator and padding

	// Get file list content
	listContent := m.list.View()
	listLines := strings.Split(listContent, "\n")

	// Get currently selected file from list
	var currentSelectedIdx int
	if len(m.list.Items()) > 0 {
		selectedItem := m.list.SelectedItem()
		if selectedItem != nil {
			if item, ok := selectedItem.(fileItem); ok {
				currentSelectedIdx = item.index
			}
		}
	}

	// Get preview content for currently selected file
	var previewLines []string
	if currentSelectedIdx >= 0 && currentSelectedIdx < len(m.files) {
		previewLines = m.renderPreviewForFile(currentSelectedIdx, previewWidth)
	} else {
		previewLines = []string{"No file selected"}
	}

	// Calculate max lines
	maxLines := len(listLines)
	if len(previewLines) > maxLines {
		maxLines = len(previewLines)
	}

	// Render side-by-side
	separator := lipgloss.NewStyle().Foreground(lipgloss.Color("#3a3a3a")).Render("│")

	for i := 0; i < maxLines; i++ {
		// Left side (file list)
		var leftLine string
		if i < len(listLines) {
			leftLine = listLines[i]
		} else {
			leftLine = ""
		}
		leftLine = m.padOrTruncate(leftLine, fileListWidth)

		// Right side (preview)
		var rightLine string
		if i < len(previewLines) {
			rightLine = previewLines[i]
		} else {
			rightLine = ""
		}

		b.WriteString(leftLine)
		b.WriteString(" ")
		b.WriteString(separator)
		b.WriteString(" ")
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	// Help text
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := "space: hide preview | o/enter: open full view | j/k: navigate | /: search | tab: toggle view | q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

// renderPreviewForFile renders a compact preview of a specific file
func (m model) renderPreviewForFile(fileIdx int, width int) []string {
	if fileIdx < 0 || fileIdx >= len(m.files) {
		return []string{}
	}

	file := m.files[fileIdx]
	var lines []string

	// File header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("180")) // Muted tan/gold instead of bright pink

	status := "modified"
	if file.IsNew {
		status = "new file"
	} else if file.IsDeleted {
		status = "deleted"
	} else if file.IsRenamed {
		status = "renamed"
	}

	lines = append(lines, headerStyle.Render(fmt.Sprintf("%s: %s", status, file.NewPath)))
	lines = append(lines, "")

	// Render diff content
	lexer := m.renderer.getLexer(file.Extension)
	unchangedLineCounter := 0
	lineCount := 0
	maxLines := m.height - 8 // Leave room for header and help

	for hunkIdx, hunk := range file.Hunks {
		// Add separator between hunks to show line jumps
		if hunkIdx > 0 {
			prevHunk := file.Hunks[hunkIdx-1]
			// Calculate the line skip
			prevEnd := prevHunk.OldStart + prevHunk.OldLines - 1
			currentStart := hunk.OldStart
			linesSkipped := currentStart - prevEnd - 1

			separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			var separatorText string
			if linesSkipped > 0 {
				separatorText = fmt.Sprintf("⋯ (%d lines skipped) ⋯", linesSkipped)
			} else {
				separatorText = "⋯"
			}
			lines = append(lines, separatorStyle.Render(separatorText))
			lineCount++
		}

		for _, line := range hunk.Lines {
			if lineCount >= maxLines {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("... (press 'o' to view full diff)"))
				return lines
			}

			useAltStyle := false
			if line.Type == parser.LineUnchanged {
				useAltStyle = unchangedLineCounter%2 == 1
				unchangedLineCounter++
			}

			renderedLine := m.renderLineCompact(line, lexer, useAltStyle, width)
			lines = append(lines, renderedLine)
			lineCount++
		}
	}

	return lines
}

// renderLineCompact renders a single line in compact format for preview
func (m model) renderLineCompact(line parser.Line, lexer chroma.Lexer, useAltStyle bool, maxWidth int) string {
	var prefix string
	switch line.Type {
	case parser.LineDeleted:
		prefix = "- "
	case parser.LineAdded:
		prefix = "+ "
	case parser.LineUnchanged:
		prefix = "  "
	}

	content := line.Content
	if m.useColor && lexer != nil {
		content = m.renderer.highlightCode(content, lexer)
	}

	// Truncate if too long
	plainContent := stripAnsiLocal(prefix + content)
	if len(plainContent) > maxWidth {
		content = content[:maxWidth-3] + "..."
	}

	var lineStyle lipgloss.Style
	if m.useColor {
		switch line.Type {
		case parser.LineDeleted:
			lineStyle = m.renderer.theme.DeletedLineStyle
		case parser.LineAdded:
			lineStyle = m.renderer.theme.AddedLineStyle
		case parser.LineUnchanged:
			if useAltStyle {
				lineStyle = m.renderer.theme.UnchangedLineStyleAlt
			} else {
				lineStyle = m.renderer.theme.UnchangedLineStyle
			}
		}
		return lineStyle.Render(prefix + content)
	}
	return prefix + content
}

// padOrTruncate pads or truncates a string to the specified width
func (m model) padOrTruncate(s string, width int) string {
	plainLen := len(stripAnsiLocal(s))
	if plainLen > width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-plainLen)
}

// stripAnsiLocal removes ANSI codes from a string
func stripAnsiLocal(s string) string {
	inEscape := false
	result := strings.Builder{}
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
		} else if inEscape && s[i] == 'm' {
			inEscape = false
		} else if !inEscape {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}

func (m model) renderSearch() string {
	var b strings.Builder

	// Show search input at the top
	searchStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("180")). // Muted tan/gold instead of bright pink
		Padding(0, 1)

	b.WriteString(searchStyle.Render("Search Files"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Show filtered list
	b.WriteString(m.list.View())
	b.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := "Type to filter | Enter: confirm | Esc: cancel"
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
		Background(lipgloss.Color("#3a3a3a")). // Muted gray instead of bright purple
		Foreground(lipgloss.Color("#d0d0d0")). // Softer white
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
		b.WriteString(collapsedStyle.Render("File collapsed. Press 'space' to expand."))
	} else {
		// Render file diff
		var diffOutput strings.Builder

		// Temporarily capture stdout
		oldStdout := fmt.Sprint
		defer func() { _ = oldStdout }()

		// Render hunks
		lexer := m.renderer.getLexer(file.Extension)
		unchangedLineCounter := 0
		for hunkIdx, hunk := range file.Hunks {
			// Add separator between hunks to show line jumps
			if hunkIdx > 0 {
				prevHunk := file.Hunks[hunkIdx-1]
				// Calculate the line skip
				prevEnd := prevHunk.OldStart + prevHunk.OldLines - 1
				currentStart := hunk.OldStart
				linesSkipped := currentStart - prevEnd - 1

				separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
				var separatorText string
				if linesSkipped > 0 {
					separatorText = fmt.Sprintf("⋯ (%d lines skipped) ⋯", linesSkipped)
				} else {
					separatorText = "⋯"
				}
				diffOutput.WriteString(separatorStyle.Render(separatorText))
				diffOutput.WriteString("\n")
			}

			if m.unified {
				for _, line := range hunk.Lines {
					useAltStyle := false
					if line.Type == parser.LineUnchanged {
						useAltStyle = unchangedLineCounter%2 == 1
						unchangedLineCounter++
					}
					diffOutput.WriteString(m.renderLineDirect(line, lexer, useAltStyle))
					diffOutput.WriteString("\n")
				}
			} else {
				diffOutput.WriteString(m.renderHunkSplit(hunk, lexer))
			}
		}

		// Apply viewport scrolling
		allLines := strings.Split(diffOutput.String(), "\n")
		totalLines := len(allLines)

		// Calculate viewport height (height - title - help - padding)
		viewportHeight := m.height - 6
		if viewportHeight < 1 {
			viewportHeight = 10
		}

		// Clamp scroll offset
		maxScroll := totalLines - viewportHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		scrollOffset := m.scrollOffset
		if scrollOffset > maxScroll {
			scrollOffset = maxScroll
		}
		if scrollOffset < 0 {
			scrollOffset = 0
		}

		// Get visible lines
		endLine := scrollOffset + viewportHeight
		if endLine > totalLines {
			endLine = totalLines
		}

		visibleLines := allLines[scrollOffset:endLine]
		b.WriteString(strings.Join(visibleLines, "\n"))

		// Show scroll indicator if needed
		if totalLines > viewportHeight {
			b.WriteString("\n")
			scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
			percentage := int(float64(scrollOffset) / float64(maxScroll) * 100)
			if scrollOffset >= maxScroll {
				percentage = 100
			}
			b.WriteString(scrollInfo.Render(fmt.Sprintf("[%d%%] Line %d-%d of %d", percentage, scrollOffset+1, endLine, totalLines)))
		}
	}
	// test

	b.WriteString("\n\n")

	// Help bar
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := "j/k: scroll | h/l: prev/next file | space: collapse/expand | g/G: top/bottom | ctrl+d/u: page down/up | tab: toggle view | /: search | esc: back | q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) renderLineDirect(line parser.Line, lexer chroma.Lexer, useAltStyle bool) string {
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

	// Apply syntax highlighting to content
	content := line.Content
	if m.useColor && lexer != nil {
		content = m.renderer.highlightCode(content, lexer)
	}

	var lineStyle lipgloss.Style
	if m.useColor {
		switch line.Type {
		case parser.LineDeleted:
			lineStyle = m.renderer.theme.DeletedLineStyle
		case parser.LineAdded:
			lineStyle = m.renderer.theme.AddedLineStyle
		case parser.LineUnchanged:
			if useAltStyle {
				lineStyle = m.renderer.theme.UnchangedLineStyleAlt
			} else {
				lineStyle = m.renderer.theme.UnchangedLineStyle
			}
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

	unchangedLineCounter := 0

	for _, line := range hunk.Lines {
		leftLine := ""
		rightLine := ""
		useAltStyle := false

		switch line.Type {
		case parser.LineDeleted:
			leftLine = m.renderer.formatLine(line, columnWidth, lexer, true, false)
			rightLine = m.renderer.formatEmptyLine(columnWidth, false)
		case parser.LineAdded:
			leftLine = m.renderer.formatEmptyLine(columnWidth, false)
			rightLine = m.renderer.formatLine(line, columnWidth, lexer, false, false)
		case parser.LineUnchanged:
			useAltStyle = unchangedLineCounter%2 == 1
			leftLine = m.renderer.formatLine(line, columnWidth, lexer, true, useAltStyle)
			rightLine = m.renderer.formatLine(line, columnWidth, lexer, false, useAltStyle)
			unchangedLineCounter++
		}

		separator := m.renderer.theme.SeparatorStyle.Render("│")
		b.WriteString(fmt.Sprintf("%s %s %s\n", leftLine, separator, rightLine))
	}

	return b.String()
}
