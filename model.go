package main

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const (
	listView viewMode = iota
	detailView
)

type model struct {
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

type viewMode int

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
	}
}

func (m model) Init() tea.Cmd {
	if m.dockerCli != nil {
		return tea.Batch(loadContainers(m.dockerCli), tickEverySecond())
	}
	return tickEverySecond()
}
