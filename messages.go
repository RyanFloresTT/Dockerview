package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moby/moby/client"
)

type tickMsg time.Time

type containersLoadedMsg struct {
	containers client.ContainerListResult
	err        error
}

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second/4, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func loadContainers(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
		return containersLoadedMsg{containers: containers, err: err}
	}
}
