package main

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type viewMode int

const (
	listView viewMode = iota
	detailView
)

type model struct {
	keys            keyMap
	help            help.Model
	viewport        viewport.Model
	width           int
	height          int
	table           table.Model
	containers      client.ContainerListResult
	selectedIndex   int
	err             error
	loading         bool
	dockerCli       *client.Client
	mode            viewMode
	detailContainer *container.Summary
	currentTime     time.Time
}

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Start   key.Binding
	Stop    key.Binding
	Refresh key.Binding
	Back    key.Binding
	Quit    key.Binding
	Help    key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "details"),
	),
	Start: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "start"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Start, k.Stop, k.Refresh},
		{k.Back, k.Quit},
	}
}

func initialModel() model {
	// Create a Docker client
	cli, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return model{err: err, loading: false}
	}

	// Setup table columns
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Image", Width: 30},
		{Title: "Status", Width: 20},
		{Title: "State", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{
		dockerCli:  cli,
		table:      t,
		containers: client.ContainerListResult{},
		loading:    true,
		mode:       listView,
		keys:       keys,
		help:       help.New(),
	}
}

func (m model) Init() tea.Cmd {
	if m.dockerCli != nil {
		return tea.Batch(loadContainers(m.dockerCli), tickEverySecond())
	}
	return tickEverySecond()
}
