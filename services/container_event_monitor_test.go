package services

import (
	"sync"
	"testing"
	"time"

	"github.com/moby/moby/api/types/events"
)

func TestContainerEventTrackerRestartCoalesce(t *testing.T) {
	var mu sync.Mutex
	var logged []string
	tracker := newContainerEventTracker(func(_ int, username, action, resource, status, message string) {
		mu.Lock()
		logged = append(logged, username+":"+action+":"+resource+":"+status+":"+message)
		mu.Unlock()
	}, nil)

	containerID := "ed4f4a86a938"
	name := "api-server"

	tracker.handle(events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionKill,
		Actor:  events.Actor{ID: containerID, Attributes: map[string]string{"name": name, "image": "nginx:latest"}},
	})
	tracker.handle(events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionStart,
		Actor:  events.Actor{ID: containerID, Attributes: map[string]string{"name": name, "image": "nginx:latest"}},
	})
	tracker.handle(events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionRestart,
		Actor:  events.Actor{ID: containerID, Attributes: map[string]string{"name": name, "image": "nginx:latest"}},
	})

	mu.Lock()
	defer mu.Unlock()
	if len(logged) != 1 {
		t.Fatalf("expected 1 restart event, got %d: %v", len(logged), logged)
	}
	if logged[0] != "docker:restart:api-server:Success:Restarted via Docker host" {
		t.Fatalf("unexpected event: %q", logged[0])
	}
}

func TestContainerEventTrackerStopAfterDelay(t *testing.T) {
	var mu sync.Mutex
	var logged []string
	tracker := newContainerEventTracker(func(_ int, username, action, resource, status, message string) {
		mu.Lock()
		logged = append(logged, username+":"+action+":"+resource)
		mu.Unlock()
	}, nil)

	containerID := "abc123def456"
	tracker.handle(events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionStop,
		Actor:  events.Actor{ID: containerID, Attributes: map[string]string{"name": "worker", "image": "redis:7"}},
	})

	time.Sleep(containerStopDelay + 200*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(logged) != 1 || logged[0] != "docker:stop:worker" {
		t.Fatalf("expected delayed stop event, got %v", logged)
	}
}

func TestContainerEventTrackerRestartAction(t *testing.T) {
	var mu sync.Mutex
	var logged []string
	tracker := newContainerEventTracker(func(_ int, username, action, resource, status, message string) {
		mu.Lock()
		logged = append(logged, action+":"+resource)
		mu.Unlock()
	}, nil)

	tracker.handle(events.Message{
		Type:   events.ContainerEventType,
		Action: events.ActionRestart,
		Actor:  events.Actor{
			ID: "ed4f4a86a938",
			Attributes: map[string]string{"name": "api-server", "image": "nginx:latest"},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if len(logged) != 1 || logged[0] != "restart:api-server" {
		t.Fatalf("expected restart action, got %v", logged)
	}
}
