package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielss-dev/critica/internal/parser"
	"golang.org/x/term"
)

// Renderer handles the display of diff output
type Renderer struct {
	theme     *Theme
	useColor  bool
	unified   bool
	termWidth int
}

// NewRenderer creates a new renderer
func NewRenderer(useColor, unified bool) *Renderer {
	// Get terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width == 0 {
		width = 120 // Default width
	}

	theme := NewTheme()
	if !useColor {
		theme = NoColorTheme()
	}

	return &Renderer{
		theme:     theme,
		useColor:  useColor,
		unified:   unified,
		termWidth: width,
	}
}

// Render displays the diff for all files
func (r *Renderer) Render(files []parser.FileDiff) {
	for i, file := range files {
		if i > 0 {
			fmt.Println() // Space between files
		}
		r.renderFile(file)
	}
}

// renderFile renders a single file diff
func (r *Renderer) renderFile(file parser.FileDiff) {
	// Print file header
	header := r.formatFileHeader(file)
	fmt.Println(header)
	fmt.Println()

	// Get lexer for syntax highlighting
	lexer := r.getLexer(file.Extension)

	// Render each hunk
	for _, hunk := range file.Hunks {
		if r.unified {
			r.renderHunkUnified(hunk, lexer)
		} else {
			r.renderHunk(hunk, lexer)
		}
		fmt.Println() // Space between hunks
	}
}

// formatFileHeader creates the file header display
func (r *Renderer) formatFileHeader(file parser.FileDiff) string {
	var status string
	switch {
	case file.IsNew:
		status = "new file"
	case file.IsDeleted:
		status = "deleted"
	case file.IsRenamed:
		status = "renamed"
	default:
		status = "modified"
	}

	headerText := fmt.Sprintf(" %s: %s ", status, file.NewPath)

	if r.useColor {
		return r.theme.FileHeaderStyle.Render(headerText)
	}
	return headerText
}

// renderHunk renders a single hunk in split-screen format
func (r *Renderer) renderHunk(hunk parser.Hunk, lexer chroma.Lexer) {
	// Calculate column width (split screen)
	columnWidth := (r.termWidth - 3) / 2 // -3 for separator and padding
	if columnWidth < 40 {
		columnWidth = 40 // Minimum width
	}

	// Build left (old) and right (new) columns
	leftLines := []string{}
	rightLines := []string{}
	unchangedLineCounter := 0

	for _, line := range hunk.Lines {
		leftLine := ""
		rightLine := ""
		useAltStyle := false

		switch line.Type {
		case parser.LineDeleted:
			// Show on left only
			leftLine = r.formatLine(line, columnWidth, lexer, true, false)
			rightLine = r.formatEmptyLine(columnWidth, false)

		case parser.LineAdded:
			// Show on right only
			leftLine = r.formatEmptyLine(columnWidth, false)
			rightLine = r.formatLine(line, columnWidth, lexer, false, false)

		case parser.LineUnchanged:
			// Show on both sides with alternating style
			useAltStyle = unchangedLineCounter%2 == 1
			leftLine = r.formatLine(line, columnWidth, lexer, true, useAltStyle)
			rightLine = r.formatLine(line, columnWidth, lexer, false, useAltStyle)
			unchangedLineCounter++
		}

		leftLines = append(leftLines, leftLine)
		rightLines = append(rightLines, rightLine)
	}

	// Print split-screen output
	separator := r.theme.SeparatorStyle.Render("│")
	for i := 0; i < len(leftLines); i++ {
		fmt.Printf("%s %s %s\n", leftLines[i], separator, rightLines[i])
	}
}

// renderHunkUnified renders a single hunk in unified diff format
func (r *Renderer) renderHunkUnified(hunk parser.Hunk, lexer chroma.Lexer) {
	unchangedLineCounter := 0

	for _, line := range hunk.Lines {
		// Determine line number and prefix
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

		// Format line number
		lineNumStr := ""
		if lineNum > 0 {
			lineNumStr = fmt.Sprintf("%4d", lineNum)
		} else {
			lineNumStr = "    "
		}

		// Apply line number style
		if r.useColor {
			lineNumColor := r.theme.LineNumUnchanged
			switch line.Type {
			case parser.LineDeleted:
				lineNumColor = r.theme.LineNumDeleted
			case parser.LineAdded:
				lineNumColor = r.theme.LineNumAdded
			}
			lineNumStr = lipgloss.NewStyle().
				Foreground(lineNumColor).
				Width(4).
				Align(lipgloss.Right).
				Render(lineNumStr)
		}

		// Format content
		content := line.Content

		// Apply syntax highlighting if enabled
		if r.useColor && lexer != nil {
			content = r.highlightCode(content, lexer)
		}

		// Apply line style based on type with alternating rows
		var lineStyle lipgloss.Style
		if r.useColor {
			switch line.Type {
			case parser.LineDeleted:
				lineStyle = r.theme.DeletedLineStyle
			case parser.LineAdded:
				lineStyle = r.theme.AddedLineStyle
			case parser.LineUnchanged:
				// Alternate between two styles for unchanged lines
				if unchangedLineCounter%2 == 0 {
					lineStyle = r.theme.UnchangedLineStyle
				} else {
					lineStyle = r.theme.UnchangedLineStyleAlt
				}
				unchangedLineCounter++
			}
		} else {
			lineStyle = lipgloss.NewStyle()
		}

		// Combine line number, prefix, and content
		fullLine := lineNumStr + " " + prefix + " " + content

		if r.useColor {
			fmt.Println(lineStyle.Render(fullLine))
		} else {
			fmt.Println(fullLine)
		}
	}
}

// formatLine formats a single line with line number and content
func (r *Renderer) formatLine(line parser.Line, width int, lexer chroma.Lexer, isLeft bool, useAltStyle bool) string {
	// Get line number
	lineNum := line.OldLineNum
	if !isLeft {
		lineNum = line.NewLineNum
	}

	lineNumStr := ""
	if lineNum > 0 {
		lineNumStr = fmt.Sprintf("%4d", lineNum)
	} else {
		lineNumStr = "    "
	}

	// Apply line number style
	if r.useColor {
		lineNumColor := r.theme.LineNumUnchanged
		switch line.Type {
		case parser.LineDeleted:
			lineNumColor = r.theme.LineNumDeleted
		case parser.LineAdded:
			lineNumColor = r.theme.LineNumAdded
		}
		lineNumStr = lipgloss.NewStyle().
			Foreground(lineNumColor).
			Width(4).
			Align(lipgloss.Right).
			Render(lineNumStr)
	}

	// Format content
	content := line.Content

	// Apply syntax highlighting if enabled
	if r.useColor && lexer != nil {
		content = r.highlightCode(content, lexer)
	}

	// Truncate or pad content to fit width
	contentWidth := width - 5 // 4 for line number, 1 for space
	content = r.fitContent(content, contentWidth)

	// Apply line style based on type with alternating support
	var lineStyle lipgloss.Style
	if r.useColor {
		switch line.Type {
		case parser.LineDeleted:
			lineStyle = r.theme.DeletedLineStyle
		case parser.LineAdded:
			lineStyle = r.theme.AddedLineStyle
		case parser.LineUnchanged:
			if useAltStyle {
				lineStyle = r.theme.UnchangedLineStyleAlt
			} else {
				lineStyle = r.theme.UnchangedLineStyle
			}
		}
	} else {
		lineStyle = lipgloss.NewStyle()
	}

	// Combine line number and content
	fullLine := lineNumStr + " " + content

	if r.useColor {
		// For unchanged lines, don't apply width to avoid forced background
		if line.Type == parser.LineUnchanged {
			return lineStyle.Render(r.padRight(fullLine, width))
		}
		return lineStyle.Width(width).Render(fullLine)
	}
	return r.padRight(fullLine, width)
}

// formatEmptyLine creates an empty line for the split screen
func (r *Renderer) formatEmptyLine(width int, useAltStyle bool) string {
	emptyLine := strings.Repeat(" ", width)
	// No background for empty lines to blend with terminal
	return emptyLine
}

// fitContent truncates or pads content to fit the specified width
func (r *Renderer) fitContent(content string, width int) string {
	// Remove ANSI codes for length calculation
	plainContent := stripAnsi(content)

	if len(plainContent) > width {
		// Truncate
		return content[:width]
	}

	// Pad with spaces
	padding := width - len(plainContent)
	return content + strings.Repeat(" ", padding)
}

// padRight pads a string to the right to reach the desired width
func (r *Renderer) padRight(s string, width int) string {
	plainLen := len(stripAnsi(s))
	if plainLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-plainLen)
}

// highlightCode applies syntax highlighting to code
func (r *Renderer) highlightCode(code string, lexer chroma.Lexer) string {
	if lexer == nil {
		return code
	}

	// Tokenize
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	// Format with terminal256 formatter
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return code
	}

	// Use a dark style
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	var buf strings.Builder
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code
	}

	return strings.TrimSuffix(buf.String(), "\n")
}

// getLexer returns the appropriate lexer for the file extension
func (r *Renderer) getLexer(extension string) chroma.Lexer {
	if !r.useColor {
		return nil
	}

	// Map common extensions to lexers
	lexer := lexers.Match(extension)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	return lexer
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	// Simple ANSI code stripper
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
