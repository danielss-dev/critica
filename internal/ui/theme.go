package ui

import (
	"strings"

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

	UseLineBackground bool
}

type ThemeOptions struct {
	DiffStyle        string
	AddedTextColor   string
	DeletedTextColor string
}

// NewTheme creates a new theme with styles based on the provided options.
func NewTheme(opts ThemeOptions) *Theme {
	switch strings.ToLower(strings.TrimSpace(opts.DiffStyle)) {
	case "patch":
		return newPatchTheme(opts)
	case "filled":
		return newFilledTheme(opts)
	default:
		return newDefaultTheme(opts)
	}
}

func newFilledTheme(opts ThemeOptions) *Theme {
	deletedFg := selectColor(lipgloss.Color("#ff8ba3"), opts.DeletedTextColor)
	addedFg := selectColor(lipgloss.Color("#8df0b5"), opts.AddedTextColor)

	t := &Theme{
		DeletedBg:         lipgloss.Color("#4c2736"),
		AddedBg:           lipgloss.Color("#1f3f48"),
		UnchangedBg:       lipgloss.Color("#232631"),
		UnchangedBgStripe: lipgloss.Color("#1c1f29"),
		InlineDeletedBg:   lipgloss.Color("#6f3246"),
		InlineAddedBg:     lipgloss.Color("#255961"),
		DeletedFg:         deletedFg,
		AddedFg:           addedFg,
		UnchangedFg:       lipgloss.Color("#d7dbe4"),
		InlineDeletedFg:   lipgloss.Color("#ffeaf2"),
		InlineAddedFg:     lipgloss.Color("#e4fff4"),
		LineNumDeleted:    dimOrOverride(lipgloss.Color("#b86c7a"), opts.DeletedTextColor),
		LineNumAdded:      dimOrOverride(lipgloss.Color("#68bda1"), opts.AddedTextColor),
		LineNumUnchanged:  lipgloss.Color("#6f7688"),
		FileHeaderBg:      lipgloss.Color("#161922"),
		FileHeaderFg:      lipgloss.Color("#edf0f7"),
		BorderColor:       lipgloss.Color("#2f3541"),
		UseLineBackground: true,
	}

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

	t.InlineDeletedStyle = lipgloss.NewStyle().
		Background(t.InlineDeletedBg).
		Foreground(t.InlineDeletedFg)

	t.InlineAddedStyle = lipgloss.NewStyle().
		Background(t.InlineAddedBg).
		Foreground(t.InlineAddedFg)

	t.LineNumStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565d70")).
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

func newDefaultTheme(opts ThemeOptions) *Theme {
	deletedFg := selectColor(lipgloss.Color("#c86b6b"), opts.DeletedTextColor)
	addedFg := selectColor(lipgloss.Color("#6bc86b"), opts.AddedTextColor)

	t := &Theme{
		DeletedBg:         lipgloss.Color("#3a2020"),
		AddedBg:           lipgloss.Color("#203a20"),
		UnchangedBg:       lipgloss.Color(""),
		UnchangedBgStripe: lipgloss.Color(""),
		InlineDeletedBg:   lipgloss.Color("#6b2c2c"),
		InlineAddedBg:     lipgloss.Color("#2c6b2c"),
		DeletedFg:         deletedFg,
		AddedFg:           addedFg,
		UnchangedFg:       lipgloss.Color("#a0a0a0"),
		InlineDeletedFg:   lipgloss.Color("#ffeeee"),
		InlineAddedFg:     lipgloss.Color("#eeffee"),
		LineNumDeleted:    dimOrOverride(lipgloss.Color("#7a5f5f"), opts.DeletedTextColor),
		LineNumAdded:      dimOrOverride(lipgloss.Color("#5f7a5f"), opts.AddedTextColor),
		LineNumUnchanged:  lipgloss.Color("#4a4a4a"),
		FileHeaderBg:      lipgloss.Color("#2a2a2a"),
		FileHeaderFg:      lipgloss.Color("#d0d0d0"),
		BorderColor:       lipgloss.Color("#333333"),
		UseLineBackground: true,
	}

	t.DeletedLineStyle = lipgloss.NewStyle().
		Background(t.DeletedBg).
		Foreground(t.DeletedFg).
		Bold(true)

	t.AddedLineStyle = lipgloss.NewStyle().
		Background(t.AddedBg).
		Foreground(t.AddedFg).
		Bold(true)

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

func newPatchTheme(opts ThemeOptions) *Theme {
	deletedFg := selectColor(lipgloss.Color("#ff6b6b"), opts.DeletedTextColor)
	addedFg := selectColor(lipgloss.Color("#6bff95"), opts.AddedTextColor)

	t := &Theme{
		DeletedBg:         lipgloss.Color(""),
		AddedBg:           lipgloss.Color(""),
		UnchangedBg:       lipgloss.Color(""),
		UnchangedBgStripe: lipgloss.Color(""),
		InlineDeletedBg:   lipgloss.Color("#ff6363"),
		InlineAddedBg:     lipgloss.Color("#34d399"),
		DeletedFg:         deletedFg,
		AddedFg:           addedFg,
		UnchangedFg:       lipgloss.Color("#c0c0c0"),
		InlineDeletedFg:   lipgloss.Color("#1f2937"),
		InlineAddedFg:     lipgloss.Color("#1f2937"),
		LineNumDeleted:    deletedFg,
		LineNumAdded:      addedFg,
		LineNumUnchanged:  lipgloss.Color("#6b7280"),
		FileHeaderBg:      lipgloss.Color("#2a2a2a"),
		FileHeaderFg:      lipgloss.Color("#d0d0d0"),
		BorderColor:       lipgloss.Color("#333333"),
		UseLineBackground: false,
	}

	t.DeletedLineStyle = lipgloss.NewStyle().
		Foreground(t.DeletedFg).
		Bold(true)

	t.AddedLineStyle = lipgloss.NewStyle().
		Foreground(t.AddedFg).
		Bold(true)

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

func selectColor(defaultColor lipgloss.Color, override string) lipgloss.Color {
	if override != "" {
		return lipgloss.Color(override)
	}
	return defaultColor
}

func dimOrOverride(defaultColor lipgloss.Color, override string) lipgloss.Color {
	if override != "" {
		return lipgloss.Color(override)
	}
	return defaultColor
}

// NoColorTheme creates a theme without colors
func NoColorTheme() *Theme {
	t := &Theme{UseLineBackground: false}

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
