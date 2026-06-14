package audit

import (
	"strings"
	"sync"
	"time"
)

type suppressKey struct {
	containerID string
	action      string
}

var (
	suppressMu sync.Mutex
	suppress   = map[suppressKey]time.Time{}
)

// SuppressContainerEvent skips matching Docker event notifications for a short window.
// Used when DockLog already logged the same container action.
func SuppressContainerEvent(containerID, action string, window time.Duration) {
	containerID = normalizeContainerID(containerID)
	if containerID == "" || window <= 0 {
		return
	}

	until := time.Now().Add(window)
	suppressMu.Lock()
	defer suppressMu.Unlock()

	suppress[suppressKey{containerID, action}] = until
	if action == "restart" {
		suppress[suppressKey{containerID, "start"}] = until
		suppress[suppressKey{containerID, "stop"}] = until
	}
}

// ShouldSuppressContainerEvent reports whether a Docker event notification should be skipped.
func ShouldSuppressContainerEvent(containerID, action string) bool {
	containerID = normalizeContainerID(containerID)
	if containerID == "" {
		return false
	}

	suppressMu.Lock()
	defer suppressMu.Unlock()

	key := suppressKey{containerID, action}
	until, ok := suppress[key]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(suppress, key)
		return false
	}
	return true
}

func normalizeContainerID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
