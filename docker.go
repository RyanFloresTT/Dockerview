package main

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func RestartContainer(m model, d container.Summary) (client.ContainerRestartResult, error) {
	restart, err := m.dockerCli.ContainerRestart(context.Background(), d.ID, client.ContainerRestartOptions{})
	if err != nil {
		return restart, err
	}
	return restart, nil
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
