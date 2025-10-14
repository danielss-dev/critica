package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielss-dev/critica/internal/parser"
	"golang.org/x/term"
)

// RendererOptions describes configuration for the renderer visuals.
type RendererOptions struct {
	UseColor         bool
	Unified          bool
	DiffStyle        string
	AddedTextColor   string
	DeletedTextColor string
}

// Renderer handles the display of diff output
type Renderer struct {
	theme     *Theme
	useColor  bool
	unified   bool
	termWidth int
}

type inlineSegment struct {
	text    string
	changed bool
}

// NewRenderer creates a new renderer
func NewRenderer(opts RendererOptions) *Renderer {
	// Get terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width == 0 {
		width = 120 // Default width
	}

	theme := NewTheme(ThemeOptions{
		DiffStyle:        opts.DiffStyle,
		AddedTextColor:   opts.AddedTextColor,
		DeletedTextColor: opts.DeletedTextColor,
	})
	if !opts.UseColor {
		theme = NoColorTheme()
	}

	return &Renderer{
		theme:     theme,
		useColor:  opts.UseColor,
		unified:   opts.Unified,
		termWidth: width,
	}
}

func computeLinePairs(lines []parser.Line) map[int]string {
	pairs := make(map[int]string)

	for i := 0; i < len(lines)-1; i++ {
		current := lines[i]
		next := lines[i+1]
		if current.Type == parser.LineDeleted && next.Type == parser.LineAdded {
			pairs[i] = next.Content
			pairs[i+1] = current.Content
			i++
		}
	}

	return pairs
}

func splitInlineSegments(text, counterpart string) []inlineSegment {
	if counterpart == "" {
		return nil
	}

	textRunes := []rune(text)
	counterRunes := []rune(counterpart)

	prefix := commonPrefixLength(textRunes, counterRunes)

	remainingText := textRunes[prefix:]
	remainingCounter := counterRunes[prefix:]

	suffix := commonSuffixLength(remainingText, remainingCounter)
	if suffix > len(remainingText) {
		suffix = len(remainingText)
	}

	var segments []inlineSegment

	if prefix > 0 {
		segments = append(segments, inlineSegment{text: string(textRunes[:prefix])})
	}

	changedEnd := len(textRunes) - suffix
	if changedEnd < prefix {
		changedEnd = prefix
	}

	if changedEnd > prefix {
		segments = append(segments, inlineSegment{text: string(textRunes[prefix:changedEnd]), changed: true})
	}

	if suffix > 0 && changedEnd < len(textRunes) {
		segments = append(segments, inlineSegment{text: string(textRunes[changedEnd:])})
	}

	if len(segments) == 0 {
		return nil
	}

	return segments
}

func commonPrefixLength(a, b []rune) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	count := 0
	for count < limit && a[count] == b[count] {
		count++
	}
	return count
}

func commonSuffixLength(a, b []rune) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	count := 0
	for count < limit && a[len(a)-1-count] == b[len(b)-1-count] {
		count++
	}
	return count
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
	for hunkIdx, hunk := range file.Hunks {
		// Add separator between hunks to show line jumps
		if hunkIdx > 0 {
			prevHunk := file.Hunks[hunkIdx-1]

			// Calculate the line skip
			// Previous hunk ends at: OldStart + OldLines - 1
			prevEnd := prevHunk.OldStart + prevHunk.OldLines - 1
			// Current hunk starts at: OldStart
			currentStart := hunk.OldStart
			linesSkipped := currentStart - prevEnd - 1

			fmt.Println(r.renderSkipSeparator(r.termWidth, linesSkipped))
		}

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
	pairs := computeLinePairs(hunk.Lines)

	for idx, line := range hunk.Lines {
		pair := pairs[idx]
		leftLine := ""
		rightLine := ""
		useAltStyle := false

		switch line.Type {
		case parser.LineDeleted:
			// Show on left only
			leftLine = r.formatLine(line, columnWidth, lexer, true, false, pair)
			rightLine = r.formatEmptyLine(columnWidth)

		case parser.LineAdded:
			// Show on right only
			leftLine = r.formatEmptyLine(columnWidth)
			rightLine = r.formatLine(line, columnWidth, lexer, false, false, pair)

		case parser.LineUnchanged:
			// Show on both sides with alternating style
			useAltStyle = unchangedLineCounter%2 == 1
			leftLine = r.formatLine(line, columnWidth, lexer, true, useAltStyle, "")
			rightLine = r.formatLine(line, columnWidth, lexer, false, useAltStyle, "")
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
	pairs := computeLinePairs(hunk.Lines)

	for idx, line := range hunk.Lines {
		pair := pairs[idx]
		// Determine line number and prefix
		var lineNum int
		var prefix string
		useAltStyle := false

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
			useAltStyle = unchangedLineCounter%2 == 1
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

		// Format content with inline diff awareness
		content := r.buildLineContent(line, lexer, pair)

		// Apply line style based on type with alternating rows
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
				unchangedLineCounter++
			default:
				lineStyle = lipgloss.NewStyle()
			}
		} else {
			lineStyle = lipgloss.NewStyle()
		}

		// Combine line number, prefix, and content
		fullLine := lineNumStr + " " + prefix + " " + content
		width := r.termWidth
		textWidth := lipgloss.Width(fullLine)
		if width < textWidth {
			width = textWidth
		}
		rendered := lineStyle.Copy().Width(width).Render(fullLine)
		rendered = r.applyLineBackground(rendered, line.Type, useAltStyle)

		fmt.Println(rendered)
	}
}

func (r *Renderer) buildLineContent(line parser.Line, lexer chroma.Lexer, counterpart string) string {
	if !r.useColor {
		return line.Content
	}

	segments := splitInlineSegments(line.Content, counterpart)
	if len(segments) == 0 {
		if lexer != nil {
			return r.highlightCode(line.Content, lexer)
		}
		return line.Content
	}

	var builder strings.Builder

	for _, segment := range segments {
		if segment.text == "" {
			continue
		}

		if segment.changed {
			inlineStyle := r.theme.InlineDeletedStyle
			if line.Type == parser.LineAdded {
				inlineStyle = r.theme.InlineAddedStyle
			}
			builder.WriteString(inlineStyle.Render(segment.text))
			continue
		}

		if lexer != nil {
			builder.WriteString(r.highlightCode(segment.text, lexer))
		} else {
			builder.WriteString(segment.text)
		}
	}

	return builder.String()
}

// formatLine formats a single line with line number and content
func (r *Renderer) formatLine(line parser.Line, width int, lexer chroma.Lexer, isLeft bool, useAltStyle bool, counterpart string) string {
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

	// Format content with optional inline highlighting
	content := r.buildLineContent(line, lexer, counterpart)

	// Truncate or pad content to fit width
	contentWidth := width - 5 // 4 for line number, 1 for space
	content = r.fitContent(content, contentWidth)

	// Apply line style based on type with alternating support
	var lineStyle lipgloss.Style
	if r.useColor {
		switch line.Type {
		case parser.LineDeleted:
			lineStyle = r.theme.DeletedLineStyle.Copy().Width(width)
		case parser.LineAdded:
			lineStyle = r.theme.AddedLineStyle.Copy().Width(width)
		case parser.LineUnchanged:
			if useAltStyle {
				lineStyle = r.theme.UnchangedLineStyleAlt.Copy().Width(width)
			} else {
				lineStyle = r.theme.UnchangedLineStyle.Copy().Width(width)
			}
		default:
			lineStyle = lipgloss.NewStyle().Width(width)
		}
	} else {
		lineStyle = lipgloss.NewStyle().Width(width)
	}

	// Combine line number and content
	fullLine := lineNumStr + " " + content

	rendered := lineStyle.Render(fullLine)
	rendered = r.applyLineBackground(rendered, line.Type, useAltStyle)
	return rendered
}

// formatEmptyLine creates an empty line for the split screen
func (r *Renderer) formatEmptyLine(width int) string {
	emptyLine := strings.Repeat(" ", width)
	// No background for empty lines to blend with terminal
	return emptyLine
}

// fitContent truncates or pads content to fit the specified width
func (r *Renderer) fitContent(content string, width int) string {
	// Remove ANSI codes for length calculation
	plainContent := stripAnsi(content)

	if len(plainContent) > width {
		// Truncate: need to find the position in the styled string
		// that corresponds to 'width' visible characters
		visibleCount := 0
		inEscape := false
		truncateAt := 0

		for i := 0; i < len(content) && visibleCount < width; i++ {
			if content[i] == '\x1b' {
				inEscape = true
			} else if inEscape && content[i] == 'm' {
				inEscape = false
			} else if !inEscape {
				visibleCount++
			}
			truncateAt = i + 1
		}

		return content[:truncateAt]
	}

	// Pad with spaces
	padding := width - len(plainContent)
	return content + strings.Repeat(" ", padding)
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

func (r *Renderer) backgroundForLineType(lineType parser.LineType, useAlt bool) lipgloss.Color {
	switch lineType {
	case parser.LineDeleted:
		return r.theme.DeletedBg
	case parser.LineAdded:
		return r.theme.AddedBg
	case parser.LineUnchanged, parser.LineContext:
		if useAlt {
			return r.theme.UnchangedBgStripe
		}
		return r.theme.UnchangedBg
	default:
		return ""
	}
}

func (r *Renderer) applyLineBackground(s string, lineType parser.LineType, useAlt bool) string {
	if !r.useColor || !r.theme.UseLineBackground {
		return s
	}
	color := r.backgroundForLineType(lineType, useAlt)
	if color == "" {
		return s
	}
	return applyPersistentBackground(s, color)
}

func applyPersistentBackground(s string, color lipgloss.Color) string {
	r, g, b, ok := parseHexColor(string(color))
	if !ok {
		return s
	}

	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	resetSeq := "\x1b[49m"

	var builder strings.Builder
	builder.Grow(len(s) + len(bgSeq)*4)
	builder.WriteString(bgSeq)

	inEscape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		builder.WriteByte(ch)
		if ch == '\x1b' {
			inEscape = true
		} else if inEscape && ch == 'm' {
			inEscape = false
			if i < len(s)-1 {
				builder.WriteString(bgSeq)
			}
		}
	}

	builder.WriteString(resetSeq)
	return builder.String()
}

func (r *Renderer) renderSkipSeparator(width int, linesSkipped int) string {
	separatorText := "⋯"
	if linesSkipped > 0 {
		separatorText = fmt.Sprintf("⋯ (%d lines skipped) ⋯", linesSkipped)
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	textWidth := lipgloss.Width(separatorText)
	if width < textWidth {
		width = textWidth
	}
	if width > 0 {
		style = style.Copy().Width(width).Align(lipgloss.Center)
	}

	return style.Render(separatorText)
}

func parseHexColor(value string) (int, int, int, bool) {
	if len(value) == 0 {
		return 0, 0, 0, false
	}
	if value[0] == '#' {
		value = value[1:]
	}
	if len(value) != 6 {
		return 0, 0, 0, false
	}

	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}

	r := int((n >> 16) & 0xff)
	g := int((n >> 8) & 0xff)
	b := int(n & 0xff)
	return r, g, b, true
}
