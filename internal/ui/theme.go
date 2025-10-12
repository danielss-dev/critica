package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme contains all color and style definitions
type Theme struct {
	// Background colors
	DeletedBg         lipgloss.Color
	AddedBg           lipgloss.Color
	UnchangedBg       lipgloss.Color
	UnchangedBgStripe lipgloss.Color // Alternating row color
	InlineDeletedBg   lipgloss.Color
	InlineAddedBg     lipgloss.Color

	// Foreground colors
	DeletedFg       lipgloss.Color
	AddedFg         lipgloss.Color
	UnchangedFg     lipgloss.Color
	InlineDeletedFg lipgloss.Color
	InlineAddedFg   lipgloss.Color

	// Line number colors
	LineNumDeleted   lipgloss.Color
	LineNumAdded     lipgloss.Color
	LineNumUnchanged lipgloss.Color

	// Header colors
	FileHeaderBg lipgloss.Color
	FileHeaderFg lipgloss.Color

	// Border colors
	BorderColor lipgloss.Color

	// Styles
	DeletedLineStyle      lipgloss.Style
	AddedLineStyle        lipgloss.Style
	UnchangedLineStyle    lipgloss.Style
	UnchangedLineStyleAlt lipgloss.Style // Alternating style
	InlineDeletedStyle    lipgloss.Style
	InlineAddedStyle      lipgloss.Style
	LineNumStyle          lipgloss.Style
	FileHeaderStyle       lipgloss.Style
	SeparatorStyle        lipgloss.Style
}

// NewTheme creates a new theme with default colors
func NewTheme() *Theme {
	t := &Theme{
		// Background colors - richer to emphasize changes
		DeletedBg:         lipgloss.Color("#3a2020"), // Very subtle dark red
		AddedBg:           lipgloss.Color("#203a20"), // Very subtle dark green
		UnchangedBg:       lipgloss.Color(""),        // Transparent/default terminal bg
		UnchangedBgStripe: lipgloss.Color("#1a1a1a"),
		InlineDeletedBg:   lipgloss.Color("#6b2c2c"),
		InlineAddedBg:     lipgloss.Color("#2c6b2c"),

		// Foreground colors - keep muted but legible
		DeletedFg:       lipgloss.Color("#c86b6b"),
		AddedFg:         lipgloss.Color("#6bc86b"),
		UnchangedFg:     lipgloss.Color("#a0a0a0"),
		InlineDeletedFg: lipgloss.Color("#ffeeee"),
		InlineAddedFg:   lipgloss.Color("#eeffee"),

		// Line numbers - very subtle
		LineNumDeleted:   lipgloss.Color("#7a5f5f"),
		LineNumAdded:     lipgloss.Color("#5f7a5f"),
		LineNumUnchanged: lipgloss.Color("#4a4a4a"),

		// Headers
		FileHeaderBg: lipgloss.Color("#2a2a2a"),
		FileHeaderFg: lipgloss.Color("#d0d0d0"),

		// Borders
		BorderColor: lipgloss.Color("#333333"),
	}

	// Create styles
	t.DeletedLineStyle = lipgloss.NewStyle().
		Background(t.DeletedBg).
		Foreground(t.DeletedFg).
		Bold(true)

	t.AddedLineStyle = lipgloss.NewStyle().
		Background(t.AddedBg).
		Foreground(t.AddedFg).
		Bold(true)

	// Unchanged lines have no background to blend with terminal
	t.UnchangedLineStyle = lipgloss.NewStyle().
		Foreground(t.UnchangedFg)

	t.UnchangedLineStyleAlt = lipgloss.NewStyle().
		Foreground(t.UnchangedFg)

	t.InlineDeletedStyle = lipgloss.NewStyle().
		Background(t.InlineDeletedBg).
		Foreground(t.InlineDeletedFg).
		Bold(true)

	t.InlineAddedStyle = lipgloss.NewStyle().
		Background(t.InlineAddedBg).
		Foreground(t.InlineAddedFg).
		Bold(true)

	t.LineNumStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Width(5).
		Align(lipgloss.Right)

	t.FileHeaderStyle = lipgloss.NewStyle().
		Background(t.FileHeaderBg).
		Foreground(t.FileHeaderFg).
		Bold(true).
		Padding(0, 1)

	t.SeparatorStyle = lipgloss.NewStyle().
		Foreground(t.BorderColor)

	return t
}

// NoColorTheme creates a theme without colors
func NoColorTheme() *Theme {
	t := &Theme{}

	t.DeletedLineStyle = lipgloss.NewStyle()
	t.AddedLineStyle = lipgloss.NewStyle()
	t.UnchangedLineStyle = lipgloss.NewStyle()
	t.UnchangedLineStyleAlt = lipgloss.NewStyle()
	t.InlineDeletedStyle = lipgloss.NewStyle()
	t.InlineAddedStyle = lipgloss.NewStyle()
	t.LineNumStyle = lipgloss.NewStyle().Width(5).Align(lipgloss.Right)
	t.FileHeaderStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	t.SeparatorStyle = lipgloss.NewStyle()

	return t
}
