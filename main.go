package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
)

type viewMode int
type tickMsg time.Time

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

type containersLoadedMsg struct {
	containers client.ContainerListResult
	err        error
}

func statusBar(width int, left, right string) string {
	bgColor := lipgloss.Color("#1e1e2e")
	fgColor := lipgloss.Color("#cdd6f4")

	style := lipgloss.NewStyle().
		Foreground(fgColor).
		Background(bgColor).
		Padding(0, 2)

	leftRendered := style.Render(left)
	rightRendered := style.Render(right)

	lWidth := lipgloss.Width(leftRendered)
	rWidth := lipgloss.Width(rightRendered)

	space := width - lWidth - rWidth
	if space < 0 {
		space = 0
	}

	spacer := lipgloss.NewStyle().
		Background(bgColor).
		Render(strings.Repeat(" ", space))

	return lipgloss.NewStyle().
		Width(width).
		Background(bgColor).
		Render(leftRendered + spacer + rightRendered)
}

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second/2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
		return tea.Batch(loadContainers(m.dockerCli), tickEverySecond())
	}
	return tickEverySecond()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.currentTime = time.Now()
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 1
		tableHeight := max(1, m.viewport.Height-5)
		m.table.SetHeight(tableHeight)
		availableWidth := max(0, m.width-2-3-20-15)

		nameWidth := int(float64(availableWidth) * 0.4)
		imageWidth := availableWidth - nameWidth

		m.table.SetColumns([]table.Column{
			{Title: "Name", Width: nameWidth},
			{Title: "Image", Width: imageWidth},
			{Title: "Status", Width: 20},
			{Title: "State", Width: 10},
		})
		return m, nil

	case tickMsg:
		m.currentTime = time.Time(msg)
		return m, tea.Batch(loadContainers(m.dockerCli), tickEverySecond())

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

			rows = append(rows, table.Row{name, c.Image, status, state})
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

			case "s":
				selectedIdx := m.table.Cursor()
				if selectedIdx >= 0 && selectedIdx < len(m.containers.Items) {
					m.detailContainer = &m.containers.Items[selectedIdx]
				}
				_, err := StopContainer(m, *m.detailContainer)
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

func StopContainer(m model, d container.Summary) (client.ContainerStopResult, error) {
	stop, err := m.dockerCli.ContainerStop(context.Background(), d.ID, client.ContainerStopOptions{})
	if err != nil {
		return stop, err
	}
	return stop, nil
}

func StartContainer(m model, d container.Summary) (client.ContainerStartResult, error) {
	start, err := m.dockerCli.ContainerStart(context.Background(), d.ID, client.ContainerStartOptions{})
	if err != nil {
		return start, err
	}
	return start, nil
}

func (m model) renderListView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 1).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	var b strings.Builder

	b.WriteString(titleStyle.Render("DockerView"))
	b.WriteString("\n\n")

	if len(m.containers.Items) == 0 {
		b.WriteString("No containers found.\n")
		b.WriteString(fmt.Sprintf("Debug: Loaded %d containers\n", len(m.containers.Items)))
		b.WriteString("\nRun 'docker ps -a' to check if you have containers.\n")
		b.WriteString("Press 'r' to refresh or 'q' to quit.\n")
	} else {
		// The baseStyle now correctly wraps just the table
		b.WriteString(baseStyle.Render(m.table.View()))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • enter details • x start • s stop • r refresh • q quit"))

	return b.String()
}

func (m model) View() string {
	// Always set viewport size
	m.viewport.Width = m.width
	m.viewport.Height = m.height - 1

	var viewContent string

	// Error state
	if m.err != nil {
		viewContent = fmt.Sprintf("\nError connecting to Docker: %v\n\nPress q to quit.\n", m.err)
	} else if m.loading {
		viewContent = "\nLoading containers...\n"
	} else if m.mode == detailView && m.detailContainer != nil {
		viewContent = m.renderDetailView()
	} else {
		viewContent = m.renderListView()
	}

	m.viewport.SetContent(viewContent)

	var left, right string
	if m.mode == detailView && m.detailContainer != nil {
		name := "unnamed"
		if len(m.detailContainer.Names) > 0 {
			name = strings.TrimPrefix(m.detailContainer.Names[0], "/")
		}
		left = fmt.Sprintf("Detail: %s", name)
	} else {
		left = fmt.Sprintf("%d containers", len(m.containers.Items))
		right = fmt.Sprintf("Selected %d  •  %s", m.table.Cursor(), m.currentTime.Format("15:04:05"))
	}

	bar := statusBar(m.width, left, right)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		bar,
	)
}

func (m model) renderDetailView() string {
	c := m.detailContainer

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
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

	b.WriteString(titleStyle.Render(fmt.Sprintf("%s", name)))
	b.WriteString("\n")

	// Container ID
	b.WriteString(labelStyle.Render("ID: "))
	b.WriteString(valueStyle.Render(c.ID[:12]))
	b.WriteString("\n")

	// Image
	b.WriteString(labelStyle.Render("Image: "))
	b.WriteString(valueStyle.Render(c.Image))
	b.WriteString("\n")

	// Image ID
	b.WriteString(labelStyle.Render("Image ID: "))
	b.WriteString(valueStyle.Render(c.ImageID))
	b.WriteString("\n")

	// Command
	b.WriteString(labelStyle.Render("Command: "))
	b.WriteString(valueStyle.Render(c.Command))
	b.WriteString("\n")

	// State
	b.WriteString(labelStyle.Render("State: "))
	stateColor := lipgloss.Color("#FF6B6B")
	if c.State == "running" {
		stateColor = "#04B575"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(stateColor).Bold(true).Render(c.State))
	b.WriteString("\n")

	// Status
	b.WriteString(labelStyle.Render("Status: "))
	b.WriteString(valueStyle.Render(c.Status))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Created: "))
	createdTime := time.Unix(c.Created, 0)
	createdString := createdTime.Format("2006-01-02 15:04:05")
	b.WriteString(valueStyle.Render(createdString))
	b.WriteString("\n")

	enumeratorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).MarginRight(1)
	rootStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("35"))
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	// Ports
	if len(c.Ports) > 0 {
		t := tree.
			Root("⁜ Ports").
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(enumeratorStyle).
			RootStyle(rootStyle).
			ItemStyle(itemStyle)

		for _, port := range c.Ports {
			if port.PublicPort > 0 {
				t.Child(fmt.Sprintf("%s:%d -> %d/%s",
					port.IP, port.PublicPort, port.PrivatePort, port.Type))
			} else {
				t.Child(fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
			}
		}
		b.WriteString("\n")
		b.WriteString(t.String())
	}

	// Networks
	if len(c.NetworkSettings.Networks) > 0 {
		t := tree.
			Root("⁜ Networks").
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(enumeratorStyle).
			RootStyle(rootStyle).
			ItemStyle(itemStyle)

		for name, network := range c.NetworkSettings.Networks {
			t.Child(fmt.Sprintf("%s (IP: %s)", name, network.IPAddress))
		}

		b.WriteString("\n\n")
		b.WriteString(t.String())
	}

	// Mounts
	if len(c.Mounts) > 0 {
		t := tree.
			Root("⁜ Mounts").
			Enumerator(tree.RoundedEnumerator).
			EnumeratorStyle(enumeratorStyle).
			RootStyle(rootStyle).
			ItemStyle(itemStyle)

		for _, mount := range c.Mounts {
			mountNode := tree.
				Root(fmt.Sprintf("🖿 %s", mount.Destination)).
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(enumeratorStyle).
				RootStyle(itemStyle).
				ItemStyle(itemStyle)

			mountNode.Child(fmt.Sprintf("Source: %s", mount.Source))
			mountNode.Child(fmt.Sprintf("Type: %s", mount.Type))

			mode := "rw"
			if !mount.RW {
				mode = "ro"
			}
			mountNode.Child(fmt.Sprintf("Mode: %s", mode))

			if mount.Propagation != "" {
				mountNode.Child(fmt.Sprintf("Propagation: %s", mount.Propagation))
			}

			t.Child(mountNode)
		}
		b.WriteString("\n\n")
		b.WriteString(t.String())
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
