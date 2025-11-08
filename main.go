package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/moby/moby/client"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
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

	return model{
		dockerCli:  cli,
		containers: client.ContainerListResult{},
		loading:    true,
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
	switch msg := msg.(type) {

	case containersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.containers = msg.containers
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

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.containers.Items)-1 {
				m.cursor++
			}

		case "r":
			m.loading = true
			return m, loadContainers(m.dockerCli)
		}
	}

	return m, nil
}

func (m model) View() string {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		PaddingTop(1).
		PaddingBottom(1)

	containerStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		MarginBottom(0)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		PaddingLeft(2)

	runningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575"))

	stoppedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B"))

	// Error state
	if m.err != nil {
		return fmt.Sprintf("\n❌ Error connecting to Docker: %v\n\nPress q to quit.\n", m.err)
	}

	// Loading state
	if m.loading {
		return "\n⏳ Loading containers...\n"
	}

	// Build the UI
	var b strings.Builder

	b.WriteString(titleStyle.Render("🐳 Docker Container Viewer"))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render(fmt.Sprintf("Found %d containers:", len(m.containers.Items))))
	b.WriteString("\n\n")

	if len(m.containers.Items) == 0 {
		b.WriteString("  No containers found.\n")
	} else {
		for i, c := range m.containers.Items {
			cursor := " "
			style := containerStyle

			if m.cursor == i {
				cursor = "▶"
				style = selectedStyle
			}

			// Container name
			name := "unnamed"
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
				if len(name) > 20 {
					name = name[:17] + "..."
				} else {
					for len(name) < 20 {
						name += " "
					}
				}
			}

			// Status styling
			status := c.Status
			if c.State == "running" {
				status = runningStyle.Render("● " + status)
			} else {
				status = stoppedStyle.Render("○ " + status)
			}

			// Format image name (shortened)
			image := c.Image
			if len(image) > 12 {
				image = image[:9] + "..."
			}

			line := fmt.Sprintf("%s %s\t    Image: %s\t    %s",
				cursor,
				name,
				image,
				status,
			)

			b.WriteString(style.Render(line))
			b.WriteString("\n\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("↑/k up • ↓/j down • r refresh • q quit"))
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
