package services

import (
	"sync"
	"time"
)

type slidingCounter struct {
	mu      sync.Mutex
	window  time.Duration
	entries []time.Time
}

func newSlidingCounter(window time.Duration) *slidingCounter {
	return &slidingCounter{window: window}
}

func (c *slidingCounter) add(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	cutoff := now.Add(-c.window)
	kept := make([]time.Time, 0, len(c.entries)+1)
	for _, ts := range c.entries {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, now)
	c.entries = kept
	return len(c.entries)
}

func (c *slidingCounter) count(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	cutoff := now.Add(-c.window)
	kept := make([]time.Time, 0, len(c.entries))
	for _, ts := range c.entries {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	c.entries = kept
	return len(c.entries)
}

type occurrenceTracker struct {
	mu    sync.Mutex
	items map[string]*slidingCounter
}

func newOccurrenceTracker() *occurrenceTracker {
	return &occurrenceTracker{items: make(map[string]*slidingCounter)}
}

func (t *occurrenceTracker) key(ruleKey, container string) string {
	return ruleKey + "|" + container
}

func (t *occurrenceTracker) add(ruleKey, container string, window time.Duration, now time.Time) int {
	key := t.key(ruleKey, container)
	t.mu.Lock()
	counter, ok := t.items[key]
	if !ok || counter.window != window {
		counter = newSlidingCounter(window)
		t.items[key] = counter
	}
	t.mu.Unlock()
	return counter.add(now)
}

func (t *occurrenceTracker) reset(ruleKey, container string) {
	t.mu.Lock()
	delete(t.items, t.key(ruleKey, container))
	t.mu.Unlock()
}
