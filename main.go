package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type viewMode int

const (
	listView viewMode = iota
	detailView
)

type model struct {
	table           table.Model
	containers      client.ContainerListResult
	selectedIndex   int
	err             error
	loading         bool
	dockerCli       *client.Client
	mode            viewMode
	detailContainer *container.Summary
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
		table.WithHeight(15),
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
		// Global keys
		switch msg.String() {
		case "ctrl+c", "q":
			if m.dockerCli != nil {
				err := m.dockerCli.Close()
				if err != nil {
					return nil, nil
				}
			}
			return m, tea.Quit
		}

		if m.mode == detailView {
			switch msg.String() {
			case "esc", "backspace":
				m.mode = listView
				m.detailContainer = nil
				return m, nil
			}
		} else {
			switch msg.String() {
			case "enter":
				if len(m.containers.Items) > 0 {
					selectedIdx := m.table.Cursor()
					if selectedIdx >= 0 && selectedIdx < len(m.containers.Items) {
						m.mode = detailView
						m.detailContainer = &m.containers.Items[selectedIdx]
						return m, nil
					}
				}

			case "r":
				m.loading = true
				return m, loadContainers(m.dockerCli)

			case "x":
				selectedIdx := m.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(m.containers.Items) {
					m.detailContainer = &m.containers.Items[selectedIdx]
				}
				_, err := StartContainer(m, *m.detailContainer)
				if err != nil {
					return nil, nil
				}

				return m, nil
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func StartContainer(m model, d container.Summary) (client.ContainerStartResult, error) {
	start, err := m.dockerCli.ContainerStart(context.Background(), d.ID, client.ContainerStartOptions{})
	if err != nil {
		return start, err
	}
	return start, nil
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

	// Detail view
	if m.mode == detailView && m.detailContainer != nil {
		return m.renderDetailView()
	}

	// List view
	var b strings.Builder

	b.WriteString(titleStyle.Render("DockerView"))
	b.WriteString("\n\n")

	if len(m.containers.Items) == 0 {
		b.WriteString("No containers found.\n")
		b.WriteString(fmt.Sprintf("Debug: Loaded %d containers\n", len(m.containers.Items)))
		b.WriteString("\nRun 'docker ps -a' to check if you have containers.\n")
		b.WriteString("Press 'r' to refresh or 'q' to quit.\n")
	} else {
		b.WriteString(fmt.Sprintf("Found %d container(s):\n\n", len(m.containers.Items)))
		b.WriteString(baseStyle.Render(m.table.View()))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • enter view details • x start container • r refresh • q quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) renderDetailView() string {
	c := m.detailContainer

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	helpStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	var b strings.Builder

	// Container name
	name := "unnamed"
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}

	b.WriteString(titleStyle.Render(fmt.Sprintf("Container Details: %s", name)))
	b.WriteString("\n\n")

	// Container ID
	b.WriteString(labelStyle.Render("ID: "))
	b.WriteString(valueStyle.Render(c.ID[:12]))
	b.WriteString("\n\n")

	// Image
	b.WriteString(labelStyle.Render("Image: "))
	b.WriteString(valueStyle.Render(c.Image))
	b.WriteString("\n\n")

	// Image ID
	b.WriteString(labelStyle.Render("Image ID: "))
	b.WriteString(valueStyle.Render(c.ImageID))
	b.WriteString("\n\n")

	// Command
	b.WriteString(labelStyle.Render("Command: "))
	b.WriteString(valueStyle.Render(c.Command))
	b.WriteString("\n\n")

	// State
	b.WriteString(labelStyle.Render("State: "))
	stateColor := lipgloss.Color("#FF6B6B")
	if c.State == "running" {
		stateColor = "#04B575"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(stateColor).Bold(true).Render(c.State))
	b.WriteString("\n\n")

	// Status
	b.WriteString(labelStyle.Render("Status: "))
	b.WriteString(valueStyle.Render(c.Status))
	b.WriteString("\n\n")

	// Created
	b.WriteString(labelStyle.Render("Created: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", c.Created)))
	b.WriteString("\n\n")

	// Ports
	if len(c.Ports) > 0 {
		b.WriteString(labelStyle.Render("Ports:\n"))
		for _, port := range c.Ports {
			if port.PublicPort > 0 {
				b.WriteString(fmt.Sprintf("  %s:%d -> %d/%s\n",
					port.IP, port.PublicPort, port.PrivatePort, port.Type))
			} else {
				b.WriteString(fmt.Sprintf("  %d/%s\n", port.PrivatePort, port.Type))
			}
		}
		b.WriteString("\n")
	}

	// Networks
	if len(c.NetworkSettings.Networks) > 0 {
		b.WriteString(labelStyle.Render("Networks:\n"))
		for name, network := range c.NetworkSettings.Networks {
			b.WriteString(fmt.Sprintf("  %s (IP: %s)\n", name, network.IPAddress))
		}
		b.WriteString("\n")
	}

	// Mounts
	if len(c.Mounts) > 0 {
		b.WriteString(labelStyle.Render("Mounts:\n"))
		for _, mount := range c.Mounts {
			b.WriteString(fmt.Sprintf("  %s -> %s (%s)\n", mount.Source, mount.Destination, mount.Type))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("esc/backspace go back • q quit"))
	b.WriteString("\n")

	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
