package services

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"docklog/audit"
	"docklog/config"
	"docklog/containers"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

const (
	containerStopDelay   = 2 * time.Second
	containerEmitDedup   = 5 * time.Second
	dockerEventUsername  = "docker"
)

type ContainerEventLogger func(userID int, username, action, resource, status, message string)

// StartContainerEventMonitor watches Docker lifecycle events and logs host CLI actions.
func StartContainerEventMonitor(cli *client.Client, onEvent ContainerEventLogger, alerts *AlertEngine) {
	if cli == nil || onEvent == nil {
		return
	}
	go runContainerEventMonitor(cli, onEvent, alerts)
}

func runContainerEventMonitor(cli *client.Client, onEvent ContainerEventLogger, alerts *AlertEngine) {
	tracker := newContainerEventTracker(onEvent, alerts)

	for {
		ctx := context.Background()
		eventFilters := client.Filters{}.Add("type", "container")

		result := cli.Events(ctx, client.EventsListOptions{Filters: eventFilters})
		streamOpen := true

		for streamOpen {
			select {
			case msg, ok := <-result.Messages:
				if !ok {
					streamOpen = false
					break
				}
				tracker.handle(msg)
			case err := <-result.Err:
				if err != nil {
					config.Debugf("Container events stream ended: %v", err)
				}
				streamOpen = false
			}
		}

		time.Sleep(2 * time.Second)
	}
}

type containerEventTracker struct {
	onEvent ContainerEventLogger
	alerts  *AlertEngine
	mu          sync.Mutex
	pending     map[string]*pendingContainerStop
	recentEmits map[string]time.Time
}

type pendingContainerStop struct {
	name  string
	image string
	timer *time.Timer
}

func newContainerEventTracker(onEvent ContainerEventLogger, alerts *AlertEngine) *containerEventTracker {
	return &containerEventTracker{
		onEvent:     onEvent,
		alerts:      alerts,
		pending:     make(map[string]*pendingContainerStop),
		recentEmits: make(map[string]time.Time),
	}
}

func (t *containerEventTracker) handle(msg events.Message) {
	if msg.Type != "" && msg.Type != events.ContainerEventType {
		return
	}

	action := strings.ToLower(string(msg.Action))
	containerID := strings.TrimSpace(msg.Actor.ID)
	name := containerNameFromEvent(msg)
	image := strings.TrimSpace(msg.Actor.Attributes["image"])

	if containerID == "" || name == "" {
		return
	}
	if containers.IsExcludedContainer(name, image) {
		return
	}

	switch action {
	case "start":
		t.handleStart(containerID, name, image)
	case "stop", "kill":
		t.scheduleStop(containerID, name, image)
	case "restart":
		t.clearPending(containerID)
		t.emit(containerID, "restart", name, "Restarted via Docker host")
		t.notifyAlerts(containerID, name, image, msg.Actor.Attributes, "restart", 0)
	case "die":
		exitCode := eventExitCode(msg)
		t.notifyAlerts(containerID, name, image, msg.Actor.Attributes, "die", exitCode)
	case "oom":
		t.notifyAlerts(containerID, name, image, msg.Actor.Attributes, "oom", 137)
	case "destroy", "remove":
		t.clearPending(containerID)
		t.emit(containerID, "remove", name, "Removed via Docker host")
	}
}

func eventExitCode(msg events.Message) int {
	raw := strings.TrimSpace(msg.Actor.Attributes["exitCode"])
	if raw == "" {
		return 1
	}
	code, err := strconv.Atoi(raw)
	if err != nil {
		return 1
	}
	return code
}

func (t *containerEventTracker) notifyAlerts(containerID, name, image string, attrs map[string]string, action string, exitCode int) {
	if t.alerts == nil {
		return
	}
	labels := map[string]string{}
	for key, value := range attrs {
		if strings.HasPrefix(key, "label.") {
			labels[strings.TrimPrefix(key, "label.")] = value
		}
	}
	t.alerts.ProcessDockerEvent(containerID, name, image, labels, action, exitCode)
	if action == "start" || action == "healthy" {
		t.alerts.EmitRecovery(containerID, name, image, labels, "healthy")
	}
}

func (t *containerEventTracker) handleStart(containerID, name, image string) {
	if containers.IsExcludedContainer(name, image) {
		return
	}

	t.mu.Lock()
	pending := t.pending[containerID]
	if pending != nil {
		delete(t.pending, containerID)
		if pending.timer != nil {
			pending.timer.Stop()
		}
		t.mu.Unlock()
		t.emit(containerID, "restart", name, "Restarted via Docker host")
		t.notifyAlerts(containerID, name, image, map[string]string{}, "restart", 0)
		return
	}
	t.mu.Unlock()

	t.emit(containerID, "start", name, "Started via Docker host")
	t.notifyAlerts(containerID, name, image, map[string]string{}, "start", 0)
}

func (t *containerEventTracker) scheduleStop(containerID, name, image string) {
	if containers.IsExcludedContainer(name, image) {
		return
	}

	t.mu.Lock()
	if existing := t.pending[containerID]; existing != nil && existing.timer != nil {
		existing.timer.Stop()
	}

	entry := &pendingContainerStop{name: name, image: image}
	entry.timer = time.AfterFunc(containerStopDelay, func() {
		t.mu.Lock()
		current := t.pending[containerID]
		if current != entry {
			t.mu.Unlock()
			return
		}
		delete(t.pending, containerID)
		t.mu.Unlock()
		t.emit(containerID, "stop", name, "Stopped via Docker host")
		t.notifyAlerts(containerID, name, image, map[string]string{}, "stop", 0)
	})
	t.pending[containerID] = entry
	t.mu.Unlock()
}

func (t *containerEventTracker) clearPending(containerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if pending := t.pending[containerID]; pending != nil && pending.timer != nil {
		pending.timer.Stop()
	}
	delete(t.pending, containerID)
}

func (t *containerEventTracker) emit(containerID, action, name, message string) {
	if audit.ShouldSuppressContainerEvent(containerID, action) {
		return
	}

	key := shortContainerID(containerID) + ":" + action
	now := time.Now()

	t.mu.Lock()
	if until, ok := t.recentEmits[key]; ok && now.Before(until) {
		t.mu.Unlock()
		return
	}
	t.recentEmits[key] = now.Add(containerEmitDedup)
	t.mu.Unlock()

	t.onEvent(0, dockerEventUsername, action, name, "Success", message)
}

func shortContainerID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func containerNameFromEvent(msg events.Message) string {
	name := strings.TrimSpace(msg.Actor.Attributes["name"])
	return strings.TrimPrefix(name, "/")
}
