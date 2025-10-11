package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme contains all color and style definitions
type Theme struct {
	// Background colors
	DeletedBg   lipgloss.Color
	AddedBg     lipgloss.Color
	UnchangedBg lipgloss.Color

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
	DeletedLineStyle   lipgloss.Style
	AddedLineStyle     lipgloss.Style
	UnchangedLineStyle lipgloss.Style
	LineNumStyle       lipgloss.Style
	FileHeaderStyle    lipgloss.Style
	SeparatorStyle     lipgloss.Style
}

// NewTheme creates a new theme with default colors
func NewTheme() *Theme {
	t := &Theme{
		// Background colors matching the screenshot
		DeletedBg:   lipgloss.Color("#3d0d0d"), // Dark red
		AddedBg:     lipgloss.Color("#0d2d0d"), // Dark green
		UnchangedBg: lipgloss.Color("#1a1a1a"), // Dark gray

		// Foreground colors
		DeletedFg:   lipgloss.Color("#ff8888"), // Light red
		AddedFg:     lipgloss.Color("#88ff88"), // Light green
		UnchangedFg: lipgloss.Color("#cccccc"), // Light gray

		// Line numbers
		LineNumDeleted:   lipgloss.Color("#ff6666"),
		LineNumAdded:     lipgloss.Color("#66ff66"),
		LineNumUnchanged: lipgloss.Color("#666666"),

		// Headers
		FileHeaderBg: lipgloss.Color("#2d2d2d"),
		FileHeaderFg: lipgloss.Color("#ffffff"),

		// Borders
		BorderColor: lipgloss.Color("#444444"),
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
