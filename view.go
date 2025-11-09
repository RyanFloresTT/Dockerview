package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/moby/moby/api/types/container"
)

func (m model) renderListView() string {
	helpStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	var b strings.Builder

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
	// Always set the viewport size
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
		left = fmt.Sprintf("Name: %s. ID: %s. Image: %s", name, m.detailContainer.ID[:12], m.detailContainer.Image)
		right = fmt.Sprintf("%s", m.currentTime.Format("15:04:05"))
	} else {
		left = fmt.Sprintf("%d containers\t● %d | ○ %d", len(m.containers.Items), GetActiveContainerCount(m.containers.Items), GetInactiveContainerCount(m.containers.Items))
		right = fmt.Sprintf("Selected %d •  %s", m.table.Cursor()+1, m.currentTime.Format("15:04:05"))
	}

	bar := statusBar(m.width, left, right)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		bar,
	)
}

func GetActiveContainerCount(c []container.Summary) int {
	count := 0
	for _, summary := range c {
		if summary.State == "running" {
			count++
		}
	}
	return count
}

func GetInactiveContainerCount(c []container.Summary) int {
	return len(c) - GetActiveContainerCount(c)
}

func (m model) renderDetailView() string {
	c := m.detailContainer

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	helpStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	var b strings.Builder

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
