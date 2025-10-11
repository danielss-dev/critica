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

	// Foreground colors
	DeletedFg   lipgloss.Color
	AddedFg     lipgloss.Color
	UnchangedFg lipgloss.Color

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
	DeletedLineStyle       lipgloss.Style
	AddedLineStyle         lipgloss.Style
	UnchangedLineStyle     lipgloss.Style
	UnchangedLineStyleAlt  lipgloss.Style // Alternating style
	LineNumStyle           lipgloss.Style
	FileHeaderStyle        lipgloss.Style
	SeparatorStyle         lipgloss.Style
}

// NewTheme creates a new theme with default colors
func NewTheme() *Theme {
	t := &Theme{
		// Background colors - more subtle and blended with terminal
		DeletedBg:         lipgloss.Color("#2d1a1a"), // Softer dark red
		AddedBg:           lipgloss.Color("#1a2d1a"), // Softer dark green
		UnchangedBg:       lipgloss.Color(""), // Transparent/default terminal bg
		UnchangedBgStripe: lipgloss.Color("#1c1c1c"), // Subtle stripe for alternating rows

		// Foreground colors - more muted
		DeletedFg:   lipgloss.Color("#d78787"), // Muted red
		AddedFg:     lipgloss.Color("#87d787"), // Muted green
		UnchangedFg: lipgloss.Color("#b2b2b2"), // Softer gray

		// Line numbers - less prominent
		LineNumDeleted:   lipgloss.Color("#8b6b6b"),
		LineNumAdded:     lipgloss.Color("#6b8b6b"),
		LineNumUnchanged: lipgloss.Color("#5f5f5f"),

		// Headers
		FileHeaderBg: lipgloss.Color("#303030"),
		FileHeaderFg: lipgloss.Color("#e0e0e0"),

		// Borders
		BorderColor: lipgloss.Color("#3a3a3a"),
	}

	// Create styles
	t.DeletedLineStyle = lipgloss.NewStyle().
		Background(t.DeletedBg).
		Foreground(t.DeletedFg)

	t.AddedLineStyle = lipgloss.NewStyle().
		Background(t.AddedBg).
		Foreground(t.AddedFg)

	t.UnchangedLineStyle = lipgloss.NewStyle().
		Background(t.UnchangedBg).
		Foreground(t.UnchangedFg)

	t.UnchangedLineStyleAlt = lipgloss.NewStyle().
		Background(t.UnchangedBgStripe).
		Foreground(t.UnchangedFg)

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
	t.LineNumStyle = lipgloss.NewStyle().Width(5).Align(lipgloss.Right)
	t.FileHeaderStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	t.SeparatorStyle = lipgloss.NewStyle()

	return t
}
