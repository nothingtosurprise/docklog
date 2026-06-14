package stats

import (
	"context"
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"docklog/db"
	"docklog/dockerutil"

	"github.com/moby/moby/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	latestSystemStats map[string]interface{}
	sysStatsMu        sync.RWMutex
	prevStats         = make(map[string]struct {
		TotalUsage  uint64
		SystemUsage uint64
	})
	prevStatsMu sync.Mutex
)

func LatestSystemStats() (map[string]interface{}, bool) {
	sysStatsMu.RLock()
	defer sysStatsMu.RUnlock()
	if latestSystemStats == nil {
		return nil, false
	}
	return latestSystemStats, true
}

func StartCollector(cli *client.Client) {
	go systemStatsBroadcaster()
	collectStats(cli)

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			collectStats(cli)
			db.DB.Exec("DELETE FROM stats WHERE timestamp < datetime('now', '-30 days')")
			db.DB.Exec("DELETE FROM system_stats WHERE timestamp < datetime('now', '-30 days')")
		}
	}()
}

func systemStatsBroadcaster() {
	for {
		v, _ := mem.VirtualMemory()
		cp, _ := cpu.Percent(500*time.Millisecond, false)
		cpuVal := 0.0
		if len(cp) > 0 {
			cpuVal = cp[0]
		}

		cores, err := cpu.Counts(true)
		if err != nil || cores == 0 {
			cores = runtime.NumCPU()
		}

		sysStatsMu.Lock()
		latestSystemStats = map[string]interface{}{
			"cpu":          cpuVal,
			"memory":       v.Used,
			"total_memory": v.Total,
			"cores":        cores,
		}
		sysStatsMu.Unlock()

		time.Sleep(500 * time.Millisecond)
	}
}

func collectStats(cli *client.Client) {
	v, _ := mem.VirtualMemory()
	cp, _ := cpu.Percent(time.Second, false)
	if len(cp) > 0 {
		db.DB.Exec("INSERT INTO system_stats (cpu, memory) VALUES (?, ?)", cp[0], v.Used)
	}

	res, _ := cli.ContainerList(context.Background(), client.ContainerListOptions{})
	containers := dockerutil.ExtractContainers(res)
	for _, ctr := range containers {
		id, _ := ctr["ID"].(string)
		if id == "" {
			id, _ = ctr["Id"].(string)
		}
		state, _ := ctr["State"].(string)
		if state != "running" {
			continue
		}
		stats, err := cli.ContainerStats(context.Background(), id, client.ContainerStatsOptions{Stream: false})
		if err != nil {
			continue
		}
		var s struct {
			CPUStats struct {
				CPUUsage struct {
					TotalUsage uint64 `json:"total_usage"`
				} `json:"cpu_usage"`
				SystemUsage uint64 `json:"system_cpu_usage"`
				OnlineCPUs  uint32 `json:"online_cpus"`
			} `json:"cpu_stats"`
			MemoryStats struct {
				Usage uint64            `json:"usage"`
				Stats map[string]uint64 `json:"stats"`
			} `json:"memory_stats"`
		}
		if err := json.NewDecoder(stats.Body).Decode(&s); err != nil {
			stats.Body.Close()
			continue
		}
		stats.Body.Close()

		cpuPercent := 0.0
		prevStatsMu.Lock()
		prev, ok := prevStats[id]
		if ok {
			cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(prev.TotalUsage)
			systemDelta := float64(s.CPUStats.SystemUsage) - float64(prev.SystemUsage)

			onlineCPUs := float64(s.CPUStats.OnlineCPUs)
			if onlineCPUs == 0 {
				onlineCPUs = float64(runtime.NumCPU())
			}

			if systemDelta > 0 && cpuDelta > 0 {
				cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
			}
		}
		prevStats[id] = struct {
			TotalUsage  uint64
			SystemUsage uint64
		}{
			TotalUsage:  s.CPUStats.CPUUsage.TotalUsage,
			SystemUsage: s.CPUStats.SystemUsage,
		}
		prevStatsMu.Unlock()

		memUsed := s.MemoryStats.Usage - (s.MemoryStats.Stats["cache"])
		db.DB.Exec("INSERT INTO stats (container_id, cpu, memory) VALUES (?, ?, ?)", id, cpuPercent, memUsed)
	}
}
