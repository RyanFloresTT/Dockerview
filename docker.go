package main

import (
	"context"
	"encoding/json"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type ContainerStats struct {
	CPUPercentage    float64
	MemoryPercentage float64
}

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
func GetContainerStats(cli *client.Client, containerID string) (*ContainerStats, error) {
	ctx := context.Background()

	stats, err := cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var v container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return nil, err
	}

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}

	memPercent := 0.0
	if v.MemoryStats.Limit > 0 {
		memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}

	return &ContainerStats{
		CPUPercentage:    cpuPercent,
		MemoryPercentage: memPercent,
	}, nil
}
