package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielss-dev/critica/internal/ai"
	"github.com/danielss-dev/critica/internal/parser"
)

type viewMode int

const (
	maxFileListPathLength = 48
)

const (
	fileListView viewMode = iota
	diffView
	searchMode
	aiAnalysisView
	aiCommitView
	aiPRView
	aiImproveView
	aiExplainView
)

type fileFilter int

const (
	filterAll fileFilter = iota
	filterStaged
	filterUnstaged
)

type model struct {
	allFiles         []parser.FileDiff
	stagedFiles      []parser.FileDiff
	unstagedFiles    []parser.FileDiff
	files            []parser.FileDiff
	fileItems        []list.Item
	list             list.Model
	textInput        textinput.Model
	viewMode         viewMode
	selectedIdx      int
	collapsed        map[int]bool
	filterMode       fileFilter
	useColor         bool
	unified          bool
	width            int
	height           int
	renderer         *Renderer
	scrollOffset     int  // Current scroll position in diff view
	previewCollapsed bool // Whether the preview pane is collapsed
	// AI-related fields
	aiService      *ai.Service
	aiResult       *ai.AnalysisResult
	aiLoading      bool
	aiError        string
	aiCommitMsg    string
	aiPRDesc       string
	aiImprovements []string
	aiExplanation  string
}

type fileItem struct {
	fullPath    string
	displayName string
	status      string
	index       int
}

func (f fileItem) FilterValue() string { return f.fullPath }
func (f fileItem) Title() string       { return f.displayName }
func (f fileItem) Description() string { return f.status }

func filterDisplayName(filter fileFilter) string {
	switch filter {
	case filterStaged:
		return "Staged"
	case filterUnstaged:
		return "Unstaged"
	default:
		return "All"
	}
}

func buildFileItems(files []parser.FileDiff) []list.Item {
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

		displayName := shortenPath(file.NewPath, maxFileListPathLength)
		items[i] = fileItem{
			fullPath:    file.NewPath,
			displayName: displayName,
			status:      status,
			index:       i,
		}
	}
	return items
}

func newCollapsedMap(length int) map[int]bool {
	collapsed := make(map[int]bool, length)
	for i := 0; i < length; i++ {
		collapsed[i] = false
	}
	return collapsed
}

func findFileIndexByPath(files []parser.FileDiff, path string) int {
	for i, file := range files {
		if file.NewPath == path {
			return i
		}
	}
	return -1
}

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

func RunInteractive(allFiles, stagedFiles, unstagedFiles []parser.FileDiff, rendererOpts RendererOptions, aiService *ai.Service) error {
	delegate := newCustomDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
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

	m := model{
		allFiles:         allFiles,
		stagedFiles:      stagedFiles,
		unstagedFiles:    unstagedFiles,
		list:             l,
		textInput:        ti,
		viewMode:         fileListView,
		selectedIdx:      -1,
		aiService:        aiService,
		filterMode:       filterAll,
		useColor:         rendererOpts.UseColor,
		unified:          rendererOpts.Unified,
		renderer:         NewRenderer(rendererOpts),
		previewCollapsed: false,
	}

	m.applyFilter(filterAll)

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *model) updateListTitle() {
	label := filterDisplayName(m.filterMode)
	m.list.Title = fmt.Sprintf("Changed Files (%s)", label)
}

func (m *model) applyFilter(filter fileFilter) {
	prevPath := ""
	if len(m.files) > 0 && m.selectedIdx >= 0 && m.selectedIdx < len(m.files) {
		prevPath = m.files[m.selectedIdx].NewPath
	}

	var target []parser.FileDiff
	switch filter {
	case filterStaged:
		target = m.stagedFiles
	case filterUnstaged:
		target = m.unstagedFiles
	default:
		target = m.allFiles
	}

	m.filterMode = filter
	m.files = target
	m.fileItems = buildFileItems(target)
	m.collapsed = newCollapsedMap(len(target))
	m.scrollOffset = 0

	m.list.SetItems(m.fileItems)

	if len(target) == 0 {
		m.selectedIdx = -1
		m.list.ResetSelected()
		m.updateListTitle()
		return
	}

	idx := -1
	if prevPath != "" {
		idx = findFileIndexByPath(target, prevPath)
	}
	if idx < 0 {
		idx = 0
	}

	m.selectedIdx = idx
	m.list.Select(idx)
	m.updateListTitle()
}

func (m *model) setFilter(filter fileFilter) {
	if m.filterMode == filter {
		return
	}
	m.applyFilter(filter)
}

func (m *model) cycleFilter() {
	next := fileFilter((int(m.filterMode) + 1) % 3)
	m.applyFilter(next)
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

			case "f":
				m.cycleFilter()
				return m, nil

			case "a":
				m.setFilter(filterAll)
				return m, nil

			case "s":
				m.setFilter(filterStaged)
				return m, nil

			case "c":
				m.setFilter(filterUnstaged)
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

			case "1":
				// AI Analysis
				if m.aiService != nil {
					m.viewMode = aiAnalysisView
					m.aiLoading = true
					m.aiError = ""
					m.scrollOffset = 0 // Reset scroll when entering AI view
					return m, m.performAIAnalysis()
				}
				return m, nil

			case "2":
				// AI Commit Message
				if m.aiService != nil {
					m.viewMode = aiCommitView
					m.aiLoading = true
					m.aiError = ""
					m.scrollOffset = 0 // Reset scroll when entering AI view
					return m, m.generateCommitMessage()
				}
				return m, nil

			case "3":
				// AI PR Description
				if m.aiService != nil {
					m.viewMode = aiPRView
					m.aiLoading = true
					m.aiError = ""
					m.scrollOffset = 0 // Reset scroll when entering AI view
					return m, m.generatePRDescription()
				}
				return m, nil

			case "4":
				// AI Improvements
				if m.aiService != nil {
					m.viewMode = aiImproveView
					m.aiLoading = true
					m.aiError = ""
					m.scrollOffset = 0 // Reset scroll when entering AI view
					return m, m.suggestImprovements()
				}
				return m, nil

			case "5":
				// AI Explain
				if m.aiService != nil {
					m.viewMode = aiExplainView
					m.aiLoading = true
					m.aiError = ""
					m.scrollOffset = 0 // Reset scroll when entering AI view
					return m, m.explainChanges()
				}
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

			case "f":
				m.cycleFilter()
				m.viewMode = fileListView
				return m, nil

			case "a":
				m.setFilter(filterAll)
				m.viewMode = fileListView
				return m, nil

			case "s":
				m.setFilter(filterStaged)
				m.viewMode = fileListView
				return m, nil

			case "c":
				m.setFilter(filterUnstaged)
				m.viewMode = fileListView
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

		case aiAnalysisView, aiCommitView, aiPRView, aiImproveView, aiExplainView:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc", "backspace":
				m.viewMode = fileListView
				m.aiLoading = false
				m.aiError = ""
				return m, nil

			case "r":
				// Retry AI operation
				if m.aiService != nil {
					m.aiLoading = true
					m.aiError = ""
					switch m.viewMode {
					case aiAnalysisView:
						return m, m.performAIAnalysis()
					case aiCommitView:
						return m, m.generateCommitMessage()
					case aiPRView:
						return m, m.generatePRDescription()
					case aiImproveView:
						return m, m.suggestImprovements()
					case aiExplainView:
						return m, m.explainChanges()
					}
				}
				return m, nil

			// Vim motions for scrolling within AI content
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
			}
		}

	// Handle AI messages
	case aiAnalysisResultMsg:
		m.aiLoading = false
		m.aiResult = msg.result
		return m, nil

	case aiAnalysisErrorMsg:
		m.aiLoading = false
		m.aiError = msg.err
		return m, nil

	case aiCommitResultMsg:
		m.aiLoading = false
		m.aiCommitMsg = msg.commitMsg
		return m, nil

	case aiCommitErrorMsg:
		m.aiLoading = false
		m.aiError = msg.err
		return m, nil

	case aiPRResultMsg:
		m.aiLoading = false
		m.aiPRDesc = msg.prDesc
		return m, nil

	case aiPRErrorMsg:
		m.aiLoading = false
		m.aiError = msg.err
		return m, nil

	case aiImproveResultMsg:
		m.aiLoading = false
		m.aiImprovements = msg.improvements
		return m, nil

	case aiImproveErrorMsg:
		m.aiLoading = false
		m.aiError = msg.err
		return m, nil

	case aiExplainResultMsg:
		m.aiLoading = false
		m.aiExplanation = msg.explanation
		return m, nil

	case aiExplainErrorMsg:
		m.aiLoading = false
		m.aiError = msg.err
		return m, nil
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
	case aiAnalysisView:
		return m.renderAIAnalysis()
	case aiCommitView:
		return m.renderAICommit()
	case aiPRView:
		return m.renderAIPR()
	case aiImproveView:
		return m.renderAIImprove()
	case aiExplainView:
		return m.renderAIExplain()
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
		if strings.Contains(strings.ToLower(fileItem.fullPath), query) {
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
		help := "space: show preview | o/enter: open full view | /: search | tab: toggle view | f: cycle filter | a: all | s: staged | c: changed | 1: AI analyze | 2: AI commit | 3: AI PR | 4: AI improve | 5: AI explain | q: quit"
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
	help := "space: hide preview | o/enter: open full view | j/k: navigate | /: search | tab: toggle view | f: cycle filter | a: all | s: staged | c: changed | 1: AI analyze | 2: AI commit | 3: AI PR | 4: AI improve | 5: AI explain | q: quit"
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

	headerWidth := width - len(status) - 2
	if headerWidth < 10 {
		headerWidth = maxFileListPathLength
	}
	displayPath := shortenPath(file.NewPath, headerWidth)
	lines = append(lines, headerStyle.Render(fmt.Sprintf("%s: %s", status, displayPath)))
	lines = append(lines, "")

	// Render diff content
	lexer := m.renderer.getLexer(file.Extension)
	unchangedLineCounter := 0
	lineCount := 0
	maxLines := m.height - 8 // Leave room for header and help
	if maxLines < 1 {
		maxLines = 5
	}

	for hunkIdx, hunk := range file.Hunks {
		if hunkIdx > 0 {
			prevHunk := file.Hunks[hunkIdx-1]
			prevEnd := prevHunk.OldStart + prevHunk.OldLines - 1
			currentStart := hunk.OldStart
			linesSkipped := currentStart - prevEnd - 1

			lines = append(lines, m.renderer.renderSkipSeparator(width, linesSkipped))
			lineCount++
		}

		pairs := computeLinePairs(hunk.Lines)
		for idx, line := range hunk.Lines {
			if lineCount >= maxLines {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("... (press 'o' to view full diff)"))
				return lines
			}

			useAltStyle := false
			if line.Type == parser.LineUnchanged {
				useAltStyle = unchangedLineCounter%2 == 1
				unchangedLineCounter++
			}

			renderedLine := m.renderLineDirect(line, lexer, useAltStyle, pairs[idx], width)
			lines = append(lines, renderedLine)
			lineCount++
		}
	}

	return lines
}

func shortenPath(path string, max int) string {
	if max <= 0 {
		return path
	}

	pathRunes := []rune(path)
	if len(pathRunes) <= max {
		return path
	}

	separator := "/"
	if strings.Contains(path, "\\") && !strings.Contains(path, "/") {
		separator = "\\"
	}

	prefix := ".../"
	if separator == "\\" {
		prefix = "...\\"
	}
	prefixRunes := []rune(prefix)

	if max <= len(prefixRunes) {
		return string(pathRunes[len(pathRunes)-max:])
	}

	segments := strings.Split(path, separator)
	if len(segments) <= 1 {
		tailLen := max - len(prefixRunes)
		if tailLen <= 0 {
			tailLen = max
		}
		return prefix + string(pathRunes[len(pathRunes)-tailLen:])
	}

	var selected []string
	total := len(prefixRunes)
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		if seg == "" {
			continue
		}

		segLen := len([]rune(seg))
		if len(selected) > 0 {
			segLen += len([]rune(separator))
		}

		if total+segLen > max && len(selected) > 0 {
			break
		}

		selected = append([]string{seg}, selected...)
		total += segLen

		if total >= max {
			break
		}
	}

	if len(selected) == 0 {
		selected = []string{segments[len(segments)-1]}
	}

	remainder := strings.Join(selected, separator)
	available := max - len(prefixRunes)
	remainderRunes := []rune(remainder)
	if available > 0 && len(remainderRunes) > available {
		remainder = string(remainderRunes[len(remainderRunes)-available:])
	}

	return prefix + remainder
}

// padOrTruncate pads or truncates a string to the specified width
func (m model) padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	plainWidth := lipgloss.Width(stripAnsiLocal(s))
	if plainWidth > width {
		trimmed, _ := truncateVisibleANSI(s, width)
		return ensureAnsiReset(trimmed)
	}
	return s + strings.Repeat(" ", width-plainWidth)
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

func truncateVisibleANSI(s string, width int) (string, bool) {
	if width <= 0 {
		return "", lipgloss.Width(stripAnsiLocal(s)) > 0
	}
	plainWidth := lipgloss.Width(stripAnsiLocal(s))
	if plainWidth <= width {
		return s, false
	}
	var builder strings.Builder
	builder.Grow(len(s))
	visible := 0
	inEscape := false
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		chunk := s[i : i+size]
		if r == '\x1b' {
			inEscape = true
			builder.WriteString(chunk)
			i += size
			continue
		}
		if inEscape {
			builder.WriteString(chunk)
			if r == 'm' {
				inEscape = false
			}
			i += size
			continue
		}

		runeWidth := lipgloss.Width(string(r))
		if visible+runeWidth > width {
			break
		}
		builder.WriteString(chunk)
		visible += runeWidth
		i += size
	}
	return builder.String(), true
}

func ensureAnsiReset(s string) string {
	const reset = "\x1b[0m"
	if !strings.Contains(s, "\x1b[") {
		return s
	}
	if strings.HasSuffix(s, reset) {
		return s
	}
	return s + reset
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

	titleWidth := m.width - 20
	if titleWidth < 20 {
		titleWidth = maxFileListPathLength
	}
	titlePath := shortenPath(file.NewPath, titleWidth)
	title := fmt.Sprintf("%s (%d/%d) - %s", titlePath, m.selectedIdx+1, len(m.files), viewMode)
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

				diffOutput.WriteString(m.renderer.renderSkipSeparator(m.width, linesSkipped))
				diffOutput.WriteString("\n")
			}

			if m.unified {
				pairs := computeLinePairs(hunk.Lines)
				for idx, line := range hunk.Lines {
					useAltStyle := false
					if line.Type == parser.LineUnchanged {
						useAltStyle = unchangedLineCounter%2 == 1
						unchangedLineCounter++
					}
					diffOutput.WriteString(m.renderLineDirect(line, lexer, useAltStyle, pairs[idx], m.width))
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

	b.WriteString("\n\n")

	// Help bar
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := "j/k: scroll | h/l: prev/next file | space: collapse/expand | g/G: top/bottom | ctrl+d/u: page down/up | tab: toggle view | f/a/s/c: filter | /: search | esc: back | q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m model) renderLineDirect(line parser.Line, lexer chroma.Lexer, useAltStyle bool, counterpart string, availableWidth int) string {
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

	content := m.renderer.buildLineContent(line, lexer, counterpart)

	fullLine := lineNumStr + " " + prefix + " " + content
	width := availableWidth
	textWidth := lipgloss.Width(fullLine)
	if width <= 0 {
		width = textWidth
	}
	if width < textWidth {
		width = textWidth
	}

	var lineStyle lipgloss.Style
	if m.useColor {
		switch line.Type {
		case parser.LineDeleted:
			lineStyle = m.renderer.theme.DeletedLineStyle.Copy().Width(width)
		case parser.LineAdded:
			lineStyle = m.renderer.theme.AddedLineStyle.Copy().Width(width)
		case parser.LineUnchanged:
			if useAltStyle {
				lineStyle = m.renderer.theme.UnchangedLineStyleAlt.Copy().Width(width)
			} else {
				lineStyle = m.renderer.theme.UnchangedLineStyle.Copy().Width(width)
			}
		default:
			lineStyle = lipgloss.NewStyle().Width(width)
		}
	} else {
		lineStyle = lipgloss.NewStyle().Width(width)
	}

	rendered := lineStyle.Render(fullLine)
	rendered = m.renderer.applyLineBackground(rendered, line.Type, useAltStyle)
	return rendered
}

func (m model) renderHunkSplit(hunk parser.Hunk, lexer chroma.Lexer) string {
	var b strings.Builder

	columnWidth := (m.width - 3) / 2
	if columnWidth < 40 {
		columnWidth = 40
	}

	unchangedLineCounter := 0
	pairs := computeLinePairs(hunk.Lines)

	for idx, line := range hunk.Lines {
		pair := pairs[idx]
		leftLine := ""
		rightLine := ""
		useAltStyle := false

		switch line.Type {
		case parser.LineDeleted:
			leftLine = m.renderer.formatLine(line, columnWidth, lexer, true, false, pair)
			rightLine = m.renderer.formatEmptyLine(columnWidth)
		case parser.LineAdded:
			leftLine = m.renderer.formatEmptyLine(columnWidth)
			rightLine = m.renderer.formatLine(line, columnWidth, lexer, false, false, pair)
		case parser.LineUnchanged:
			useAltStyle = unchangedLineCounter%2 == 1
			leftLine = m.renderer.formatLine(line, columnWidth, lexer, true, useAltStyle, "")
			rightLine = m.renderer.formatLine(line, columnWidth, lexer, false, useAltStyle, "")
			unchangedLineCounter++
		}

		separator := m.renderer.theme.SeparatorStyle.Render("│")
		b.WriteString(fmt.Sprintf("%s %s %s\n", leftLine, separator, rightLine))
	}

	return b.String()
}

// AI Command Functions

func (m *model) performAIAnalysis() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		result, err := m.aiService.AnalyzeDiff(ctx, m.files)
		if err != nil {
			return aiAnalysisErrorMsg{err.Error()}
		}
		return aiAnalysisResultMsg{result}
	}
}

func (m *model) generateCommitMessage() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		commitMsg, err := m.aiService.GenerateCommitMessage(ctx, m.files)
		if err != nil {
			return aiCommitErrorMsg{err.Error()}
		}
		return aiCommitResultMsg{commitMsg}
	}
}

func (m *model) generatePRDescription() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		prDesc, err := m.aiService.GeneratePRDescription(ctx, m.files)
		if err != nil {
			return aiPRErrorMsg{err.Error()}
		}
		return aiPRResultMsg{prDesc}
	}
}

func (m *model) suggestImprovements() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		improvements, err := m.aiService.SuggestImprovements(ctx, m.files)
		if err != nil {
			return aiImproveErrorMsg{err.Error()}
		}
		return aiImproveResultMsg{improvements}
	}
}

func (m *model) explainChanges() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		explanation, err := m.aiService.ExplainChanges(ctx, m.files)
		if err != nil {
			return aiExplainErrorMsg{err.Error()}
		}
		return aiExplainResultMsg{explanation}
	}
}

// AI Message Types

type aiAnalysisResultMsg struct {
	result *ai.AnalysisResult
}

type aiAnalysisErrorMsg struct {
	err string
}

type aiCommitResultMsg struct {
	commitMsg string
}

type aiCommitErrorMsg struct {
	err string
}

type aiPRResultMsg struct {
	prDesc string
}

type aiPRErrorMsg struct {
	err string
}

type aiImproveResultMsg struct {
	improvements []string
}

type aiImproveErrorMsg struct {
	err string
}

type aiExplainResultMsg struct {
	explanation string
}

type aiExplainErrorMsg struct {
	err string
}

// AI Render Functions

func (m model) renderAIAnalysis() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#58a6ff")).
		Margin(1, 0)

	b.WriteString(titleStyle.Render("🤖 AI Analysis Results"))
	b.WriteString("\n")

	if m.aiLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e")).
			Margin(1, 0)
		b.WriteString(loadingStyle.Render("Analyzing changes with AI..."))
		return b.String()
	}

	if m.aiError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f85149")).
			Margin(1, 0)
		b.WriteString(errorStyle.Render("Error: " + m.aiError))
		b.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e"))
		b.WriteString(helpStyle.Render("Press 'r' to retry or 'esc' to go back"))
		return b.String()
	}

	if m.aiResult == nil {
		return b.String()
	}

	// Summary
	if m.aiResult.Summary != "" {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("📝 Summary:"))
		b.WriteString("\n")
		b.WriteString(m.aiResult.Summary)
		b.WriteString("\n\n")
	}

	// Code Quality
	if m.aiResult.CodeQuality != "" {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("🏆 Code Quality:"))
		b.WriteString("\n")
		b.WriteString(m.aiResult.CodeQuality)
		b.WriteString("\n\n")
	}

	// Issues
	if len(m.aiResult.Issues) > 0 {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("⚠️  Issues Found:"))
		b.WriteString("\n")
		for i, issue := range m.aiResult.Issues {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, issue))
		}
		b.WriteString("\n")
	}

	// Improvements
	if len(m.aiResult.Improvements) > 0 {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("💡 Improvement Suggestions:"))
		b.WriteString("\n")
		for i, improvement := range m.aiResult.Improvements {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, improvement))
		}
		b.WriteString("\n")
	}

	// Security Notes
	if len(m.aiResult.SecurityNotes) > 0 {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("🔒 Security Notes:"))
		b.WriteString("\n")
		for i, note := range m.aiResult.SecurityNotes {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, note))
		}
		b.WriteString("\n")
	}

	// Performance Notes
	if len(m.aiResult.PerformanceNotes) > 0 {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("⚡ Performance Notes:"))
		b.WriteString("\n")
		for i, note := range m.aiResult.PerformanceNotes {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, note))
		}
		b.WriteString("\n")
	}

	// Commit Message
	if m.aiResult.CommitMessage != "" {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("📝 Suggested Commit Message:"))
		b.WriteString("\n")
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(0, 0, 1, 0)
		b.WriteString(codeStyle.Render(m.aiResult.CommitMessage))
		b.WriteString("\n")
	}

	// PR Description
	if m.aiResult.PRDescription != "" {
		sectionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f0f6fc")).
			Margin(0, 0, 1, 0)
		b.WriteString(sectionStyle.Render("📋 PR Description:"))
		b.WriteString("\n")
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(0, 0, 1, 0)
		b.WriteString(codeStyle.Render(m.aiResult.PRDescription))
		b.WriteString("\n")
	}

	// Apply viewport scrolling
	allLines := strings.Split(b.String(), "\n")
	totalLines := len(allLines)

	// Calculate viewport height (height - help - padding)
	viewportHeight := m.height - 4
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
	result := strings.Join(visibleLines, "\n")

	// Show scroll indicator if needed
	if totalLines > viewportHeight {
		result += "\n"
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		percentage := int(float64(scrollOffset) / float64(maxScroll) * 100)
		if scrollOffset >= maxScroll {
			percentage = 100
		}
		result += scrollInfo.Render(fmt.Sprintf("[%d%%] Line %d-%d of %d", percentage, scrollOffset+1, endLine, totalLines))
	}

	// Add help text
	result += "\n\n"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8b949e")).
		Margin(1, 0)
	result += helpStyle.Render("j/k: scroll | g/G: top/bottom | d/u: page | esc: back | r: retry")

	return result
}

func (m model) renderAICommit() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#58a6ff")).
		Margin(1, 0)

	b.WriteString(titleStyle.Render("🤖 AI Commit Message"))
	b.WriteString("\n")

	if m.aiLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e")).
			Margin(1, 0)
		b.WriteString(loadingStyle.Render("Generating commit message..."))
		return b.String()
	}

	if m.aiError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f85149")).
			Margin(1, 0)
		b.WriteString(errorStyle.Render("Error: " + m.aiError))
		b.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e"))
		b.WriteString(helpStyle.Render("Press 'r' to retry or 'esc' to go back"))
		return b.String()
	}

	// Display the actual commit message
	if m.aiCommitMsg != "" {
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(1, 0)
		b.WriteString(codeStyle.Render(m.aiCommitMsg))
	} else {
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(1, 0)
		b.WriteString(codeStyle.Render("No commit message generated"))
	}

	// Apply viewport scrolling
	allLines := strings.Split(b.String(), "\n")
	totalLines := len(allLines)

	// Calculate viewport height (height - help - padding)
	viewportHeight := m.height - 4
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
	result := strings.Join(visibleLines, "\n")

	// Show scroll indicator if needed
	if totalLines > viewportHeight {
		result += "\n"
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		percentage := int(float64(scrollOffset) / float64(maxScroll) * 100)
		if scrollOffset >= maxScroll {
			percentage = 100
		}
		result += scrollInfo.Render(fmt.Sprintf("[%d%%] Line %d-%d of %d", percentage, scrollOffset+1, endLine, totalLines))
	}

	// Add help text
	result += "\n\n"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8b949e")).
		Margin(1, 0)
	result += helpStyle.Render("j/k: scroll | g/G: top/bottom | d/u: page | esc: back | r: retry")

	return result
}

func (m model) renderAIPR() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#58a6ff")).
		Margin(1, 0)

	b.WriteString(titleStyle.Render("🤖 AI PR Description"))
	b.WriteString("\n")

	if m.aiLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e")).
			Margin(1, 0)
		b.WriteString(loadingStyle.Render("Generating PR description..."))
		return b.String()
	}

	if m.aiError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f85149")).
			Margin(1, 0)
		b.WriteString(errorStyle.Render("Error: " + m.aiError))
		b.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e"))
		b.WriteString(helpStyle.Render("Press 'r' to retry or 'esc' to go back"))
		return b.String()
	}

	// Display the actual PR description
	if m.aiPRDesc != "" {
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(1, 0)
		b.WriteString(codeStyle.Render(m.aiPRDesc))
	} else {
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(1, 0)
		b.WriteString(codeStyle.Render("No PR description generated"))
	}

	// Apply viewport scrolling
	allLines := strings.Split(b.String(), "\n")
	totalLines := len(allLines)

	// Calculate viewport height (height - help - padding)
	viewportHeight := m.height - 4
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
	result := strings.Join(visibleLines, "\n")

	// Show scroll indicator if needed
	if totalLines > viewportHeight {
		result += "\n"
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		percentage := int(float64(scrollOffset) / float64(maxScroll) * 100)
		if scrollOffset >= maxScroll {
			percentage = 100
		}
		result += scrollInfo.Render(fmt.Sprintf("[%d%%] Line %d-%d of %d", percentage, scrollOffset+1, endLine, totalLines))
	}

	// Add help text
	result += "\n\n"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8b949e")).
		Margin(1, 0)
	result += helpStyle.Render("j/k: scroll | g/G: top/bottom | d/u: page | esc: back | r: retry")

	return result
}

func (m model) renderAIImprove() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#58a6ff")).
		Margin(1, 0)

	b.WriteString(titleStyle.Render("🤖 AI Improvement Suggestions"))
	b.WriteString("\n")

	if m.aiLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e")).
			Margin(1, 0)
		b.WriteString(loadingStyle.Render("Analyzing for improvements..."))
		return b.String()
	}

	if m.aiError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f85149")).
			Margin(1, 0)
		b.WriteString(errorStyle.Render("Error: " + m.aiError))
		b.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e"))
		b.WriteString(helpStyle.Render("Press 'r' to retry or 'esc' to go back"))
		return b.String()
	}

	// Display the actual improvements
	if len(m.aiImprovements) > 0 {
		for i, improvement := range m.aiImprovements {
			itemStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f0f6fc")).
				Margin(0, 0, 1, 0)
			b.WriteString(itemStyle.Render(fmt.Sprintf("%d. %s", i+1, improvement)))
			b.WriteString("\n")
		}
	} else {
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(1, 0)
		b.WriteString(codeStyle.Render("No improvement suggestions generated"))
	}

	// Apply viewport scrolling
	allLines := strings.Split(b.String(), "\n")
	totalLines := len(allLines)

	// Calculate viewport height (height - help - padding)
	viewportHeight := m.height - 4
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
	result := strings.Join(visibleLines, "\n")

	// Show scroll indicator if needed
	if totalLines > viewportHeight {
		result += "\n"
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		percentage := int(float64(scrollOffset) / float64(maxScroll) * 100)
		if scrollOffset >= maxScroll {
			percentage = 100
		}
		result += scrollInfo.Render(fmt.Sprintf("[%d%%] Line %d-%d of %d", percentage, scrollOffset+1, endLine, totalLines))
	}

	// Add help text
	result += "\n\n"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8b949e")).
		Margin(1, 0)
	result += helpStyle.Render("j/k: scroll | g/G: top/bottom | d/u: page | esc: back | r: retry")

	return result
}

func (m model) renderAIExplain() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#58a6ff")).
		Margin(1, 0)

	b.WriteString(titleStyle.Render("🤖 AI Change Explanation"))
	b.WriteString("\n")

	if m.aiLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e")).
			Margin(1, 0)
		b.WriteString(loadingStyle.Render("Explaining changes..."))
		return b.String()
	}

	if m.aiError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f85149")).
			Margin(1, 0)
		b.WriteString(errorStyle.Render("Error: " + m.aiError))
		b.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e"))
		b.WriteString(helpStyle.Render("Press 'r' to retry or 'esc' to go back"))
		return b.String()
	}

	// Display the actual explanation
	if m.aiExplanation != "" {
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(1, 0)
		b.WriteString(codeStyle.Render(m.aiExplanation))
	} else {
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#21262d")).
			Padding(1).
			Margin(1, 0)
		b.WriteString(codeStyle.Render("No explanation generated"))
	}

	// Apply viewport scrolling
	allLines := strings.Split(b.String(), "\n")
	totalLines := len(allLines)

	// Calculate viewport height (height - help - padding)
	viewportHeight := m.height - 4
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
	result := strings.Join(visibleLines, "\n")

	// Show scroll indicator if needed
	if totalLines > viewportHeight {
		result += "\n"
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		percentage := int(float64(scrollOffset) / float64(maxScroll) * 100)
		if scrollOffset >= maxScroll {
			percentage = 100
		}
		result += scrollInfo.Render(fmt.Sprintf("[%d%%] Line %d-%d of %d", percentage, scrollOffset+1, endLine, totalLines))
	}

	// Add help text
	result += "\n\n"
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8b949e")).
		Margin(1, 0)
	result += helpStyle.Render("j/k: scroll | g/G: top/bottom | d/u: page | esc: back | r: retry")

	return result
}
