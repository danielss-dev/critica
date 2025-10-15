package layout

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Sizeable defines components that can be resized
type Sizeable interface {
	SetSize(width, height int) tea.Cmd
	GetSize() (int, int)
}

// Bindings defines components that have key bindings
type Bindings interface {
	BindingKeys() []key.Binding
}

// Container wraps a tea.Model with layout capabilities
type Container interface {
	tea.Model
	Sizeable
	Bindings
}
