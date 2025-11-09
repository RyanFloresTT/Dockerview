package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.currentTime = time.Now()
	var cmd tea.Cmd

	if m.mode == detailView && m.detailContainer != nil {
		for i := range m.containers.Items {
			if m.containers.Items[i].ID == m.detailContainer.ID {
				m.detailContainer = &m.containers.Items[i]
				break
			}
		}
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		helpHeight := lipgloss.Height(m.help.View(m.keys))

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 1 - helpHeight

		tableHeight := max(1, m.viewport.Height-5)
		m.table.SetHeight(tableHeight)

		availableWidth := max(0, m.width-2-3-20-17)

		nameWidth := int(float64(availableWidth) * 0.4)
		imageWidth := availableWidth - nameWidth

		m.table.SetColumns([]table.Column{
			{Title: "Name", Width: nameWidth},
			{Title: "Image", Width: imageWidth},
			{Title: "CPU", Width: 10},
			{Title: "Memory", Width: 10},
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

			containerStats, err := GetContainerStats(m.dockerCli, c.ID)
			if err != nil {
				continue
			}

			cpuUsage := strconv.FormatFloat(containerStats.CPUPercentage, 'f', 1, 64)
			cpuUsage += "%"
			mem := strconv.FormatFloat(containerStats.MemoryPercentage, 'f', 1, 64)
			mem += "%"

			// State indicator
			state := c.State
			if state == "running" {
				state = "● running"
			} else {
				state = "○ " + state
			}

			rows = append(rows, table.Row{name, c.Image, cpuUsage, mem, state})
		}
		m.table.SetRows(rows)
		return m, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "?":
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

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
		}

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
			selectedIdx := m.table.Cursor()
			if selectedIdx >= 0 && selectedIdx < len(m.containers.Items) {
				m.detailContainer = &m.containers.Items[selectedIdx]
			}
			_, err := RestartContainer(m, *m.detailContainer)
			if err != nil {
				return nil, nil
			}

			return m, nil

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

	if m.mode == listView {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}
