package services

import (
	"sync"
	"time"
)

type metricBreach struct {
	since time.Time
}

type metricBreachTracker struct {
	mu      sync.Mutex
	entries map[string]metricBreach
	fired   map[string]bool
}

func newMetricBreachTracker() *metricBreachTracker {
	return &metricBreachTracker{
		entries: make(map[string]metricBreach),
		fired:   make(map[string]bool),
	}
}

func (t *metricBreachTracker) key(ruleKey, container string) string {
	return ruleKey + "|" + container
}

func (t *metricBreachTracker) observe(ruleKey, container string, breached bool, duration time.Duration, now time.Time) (bool, time.Duration) {
	key := t.key(ruleKey, container)
	t.mu.Lock()
	defer t.mu.Unlock()

	if !breached {
		delete(t.entries, key)
		return false, 0
	}

	entry, ok := t.entries[key]
	if !ok {
		t.entries[key] = metricBreach{since: now}
		return false, 0
	}
	elapsed := now.Sub(entry.since)
	if elapsed >= duration {
		delete(t.entries, key)
		return true, elapsed
	}
	return false, elapsed
}

func (t *metricBreachTracker) clear(ruleKey, container string) {
	t.mu.Lock()
	delete(t.entries, t.key(ruleKey, container))
	t.mu.Unlock()
}

func (t *metricBreachTracker) markFired(ruleKey, container string) {
	t.mu.Lock()
	t.fired[t.key(ruleKey, container)] = true
	t.mu.Unlock()
}

func (t *metricBreachTracker) consumeRecovery(ruleKey, container string) bool {
	key := t.key(ruleKey, container)
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.fired[key] {
		return false
	}
	delete(t.fired, key)
	return true
}
