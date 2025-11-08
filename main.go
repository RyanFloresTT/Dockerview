package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/moby/moby/client"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table      table.Model
	containers client.ContainerListResult
	cursor     int
	err        error
	loading    bool
	dockerCli  *client.Client
}

type containersLoadedMsg struct {
	containers client.ContainerListResult
	err        error
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
		table.WithHeight(10),
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
		containers: client.ContainerListResult{},
		loading:    true,
		table:      t,
	}
}

func loadContainers(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
		return containersLoadedMsg{containers: containers, err: err}
	}
}

func (m model) Init() tea.Cmd {
	if m.dockerCli != nil {
		return loadContainers(m.dockerCli)
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case containersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.containers = msg.containers

		// Convert containers to table rows
		var rows []table.Row
		for _, c := range m.containers.Items {
			name := "unnamed"
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}

			// Truncate image if too long
			image := c.Image
			if len(image) > 28 {
				image = image[:25] + "..."
			}

			// Format status
			status := c.Status
			if len(status) > 18 {
				status = status[:15] + "..."
			}

			// State indicator
			state := c.State
			if state == "running" {
				state = "● running"
			} else {
				state = "○ " + state
			}

			rows = append(rows, table.Row{name, image, status, state})
		}
		m.table.SetRows(rows)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.dockerCli != nil {
				err := m.dockerCli.Close()
				if err != nil {
					return nil, nil
				}
			}
			return m, tea.Quit

		case "r":
			// Refresh container list
			m.loading = true
			return m, loadContainers(m.dockerCli)
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	// Error state
	if m.err != nil {
		return fmt.Sprintf("\nError connecting to Docker: %v\n\nPress q to quit.\n", m.err)
	}

	// Loading state
	if m.loading {
		return "\nLoading containers...\n"
	}

	// Build the UI
	var b strings.Builder

	b.WriteString(titleStyle.Render("Docker Container Viewer"))
	b.WriteString("\n\n")
	b.WriteString(baseStyle.Render(m.table.View()))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • r refresh • q quit"))
	b.WriteString("\n")

	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
