package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Toggle  key.Binding
	All     key.Binding
	None    key.Binding
	Clean   key.Binding
	DryRun    key.Binding
	Verbose   key.Binding
	ShowEmpty key.Binding
	Projects  key.Binding
	Refresh   key.Binding
	Help    key.Binding
	Quit    key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "categories")),
		Right:   key.NewBinding(key.WithKeys("right", "l", "tab"), key.WithHelp("→/l", "items")),
		Toggle:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
		All:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
		None:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "select none")),
		Clean:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "clean selected")),
		DryRun:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "toggle dry-run")),
		Verbose:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle verbose log")),
		ShowEmpty: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "toggle show-empty")),
		Projects:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "scan project dirs")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rescan")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Confirm: key.NewBinding(key.WithKeys("y", "Y", "enter")),
		Cancel:  key.NewBinding(key.WithKeys("n", "N", "esc")),
	}
}
