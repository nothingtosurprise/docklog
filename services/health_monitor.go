package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"docklog/containers"
	"docklog/dockerutil"

	"github.com/moby/moby/client"
)

type HealthEventLogger func(userID int, username, action, resource, status, message string)

// StartHealthMonitor watches Docker HEALTHCHECK status and logs transitions.
func StartHealthMonitor(cli *client.Client, onEvent HealthEventLogger) {
	if cli == nil || onEvent == nil {
		return
	}
	go runHealthMonitor(cli, onEvent)
}

func runHealthMonitor(cli *client.Client, onEvent HealthEventLogger) {
	known := map[string]string{}
	var mu sync.Mutex

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: false})
		if err != nil {
			continue
		}

		seen := map[string]bool{}
		for _, summary := range dockerutil.ExtractContainers(res) {
			id := containerID(summary)
			if id == "" {
				continue
			}

			name := containerName(summary)
			image, _ := summary["Image"].(string)
			if containers.IsExcludedContainer(name, image) {
				continue
			}

			health, ok := healthStatusFromSummary(summary)
			if !ok {
				continue
			}

			seen[id] = true

			mu.Lock()
			prev, hadPrev := known[id]
			known[id] = health
			mu.Unlock()

			if !hadPrev {
				continue
			}
			if prev == health {
				continue
			}

			switch health {
			case "unhealthy":
				if prev == "healthy" || prev == "starting" {
					onEvent(0, "system", "health_check_failed", name, "Error",
						"Docker health check reported unhealthy for "+name)
				}
			case "healthy":
				if prev == "unhealthy" {
					onEvent(0, "system", "health_check_recovered", name, "Success",
						"Docker health check recovered for "+name)
				}
			}
		}

		mu.Lock()
		for id := range known {
			if !seen[id] {
				delete(known, id)
			}
		}
		mu.Unlock()
	}
}

func containerID(summary map[string]interface{}) string {
	if id, ok := summary["ID"].(string); ok && id != "" {
		return id
	}
	if id, ok := summary["Id"].(string); ok {
		return id
	}
	return ""
}

func containerName(summary map[string]interface{}) string {
	names, _ := summary["Names"].([]interface{})
	if len(names) == 0 {
		return "unknown"
	}
	name, _ := names[0].(string)
	return strings.TrimPrefix(name, "/")
}

func healthStatusFromSummary(summary map[string]interface{}) (string, bool) {
	status, _ := summary["Status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(status, "(unhealthy)"):
		return "unhealthy", true
	case strings.Contains(status, "(healthy)"):
		return "healthy", true
	case strings.Contains(status, "(starting)") || strings.Contains(status, "health: starting"):
		return "starting", true
	default:
		return "", false
	}
}
